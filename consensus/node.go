package consensus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/raft"
	"github.com/icetrays/icetrays/consensus/pb"
	"github.com/icetrays/icetrays/network"
	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/go-mfs"
	gostream "github.com/libp2p/go-libp2p-gostream"
	"google.golang.org/grpc"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

type Node struct {
	raft        preCommitter
	fsm         *Fsm
	retryTimes  int
	ID          string
	bootstrapId string
	operator    Operator
	mtx         sync.Mutex
	network     *network.Network
	ctx         context.Context
	ipfs        *httpapi.HttpApi
	packer      Sender
}

func (n *Node) Op(ctx context.Context, code pb.Instruction_Code, params ...string) error {
	if n.fsm.Inconsistent() {
		return errors.New("inconsistent state")
	}

	err := n.TrySwitchOperator()
	if err != nil {
		return err
	}
	switch code {
	case pb.Instruction_CP:
		if strings.HasPrefix(params[1], "/") {
			return n.operator.Cp(ctx, params[0], params[1], nil)
		} else {
			c, err := cid.Decode(params[1])
			if err != nil {
				return err
			}
			cctx, cancel := context.WithTimeout(ctx, time.Second*20)
			defer cancel()
			ipldNode, err := n.ipfs.Dag().Get(cctx, c)
			if err != nil {
				return err
			}
			return n.operator.Cp(ctx, params[0], params[1], ipldNode.RawData())
		}
	case pb.Instruction_MV:
		return n.operator.Mv(ctx, params[0], params[1])
	case pb.Instruction_RM:
		return n.operator.Rm(ctx, params[0])
	case pb.Instruction_MKDIR:
		return n.operator.MkDir(ctx, params[0])
	default:
		return errors.New("no matched operator")
	}
}

func (n *Node) Ls(ctx context.Context, path string) ([]mfs.NodeListing, error) {
	return n.fsm.State.Ls(ctx, path)
}

func (n *Node) Leader() string {
	return string(n.raft.Leader())
}

func (n *Node) IsLeader() bool {
	return string(n.raft.Leader()) == n.ID
}

func (n *Node) Operator() string {
	return n.operator.Address()
}

func (n *Node) UploadFile(ctx context.Context, fileName string, pinCount int, read io.Reader) error {
	add, err := n.ipfs.Unixfs().Add(ctx, files.NewReaderFile(read))
	if err != nil {
		return err
	}
	fmt.Printf("fileName: %s, cid: %s", fileName, add.Cid().String())

	future := n.raft.GetConfiguration()
	r2 := BytesToBinaryString(add.Cid().String())

	counts := make([]int, 0)
	peers := make(map[int][]string)
	if err := future.Error(); err != nil {
		return err
	} else {
		servers := future.Configuration().Servers
		if pinCount > len(servers) {
			return errors.New("insufficient number of nodes")
		}
		for _, d := range servers {
			r1 := BytesToBinaryString(string(d.ID))
			l := Min(len(r1), len(r2))
			sc := 0
			for i := 0; i < l; i++ {
				if r1[i]^r2[i] == 0 {
					sc++
				}
			}
			counts = append(counts, sc)
			if t, ok := peers[sc]; !ok {
				peers[sc] = []string{string(d.ID)}
			} else {
				peers[sc] = append(t, string(d.ID))
			}
		}
	}
	sort.Ints(counts)
	rp := make([]string, 0)
	for i := len(counts) - 1; i >= 0; i-- {
		for _, peer := range peers[counts[i]] {
			pinCount--
			if pinCount < 0 {
				break
			}
			rp = append(rp, peer)
		}
	}

	pn, _ := json.Marshal(rp)
	return n.operator.PinFile(ctx, string(pn), add.Cid().String(), fileName)
}

func (n *Node) TrySwitchOperator() error {
	for {
		if n.Leader() == "" {
			time.Sleep(time.Millisecond * 20)
			continue
		}
		if n.Leader() != n.Operator() {
			if err := n.SwitchOperator(); err != nil {
				return err
			}
		}
		return nil
	}
}

func (n *Node) InitOperator() error {
	if n.bootstrapId == n.ID {
		n.operator = NewLocalOperator(n.packer, n.ID)
	} else {
		for !n.network.PeerFounded(n.bootstrapId) {
			time.Sleep(time.Millisecond * 20)
		}
		conn, err := n.network.Connect(n.ctx, n.bootstrapId)
		if err != nil {
			return err
		}
		n.operator = NewRemoteOperator(conn, n.bootstrapId)
		// todo ctx ?
		fmt.Println("add peer")
		time.Sleep(5 * time.Second)
		err = n.operator.AddPeer(context.Background(), n.ID)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	return nil
}

// todo
func (n *Node) SwitchOperator() error {
	if n.ID == n.Leader() {
		n.operator = NewLocalOperator(n.packer, n.ID)
	} else {
		conn, err := n.network.Connect(n.ctx, n.Leader())
		if err != nil {
			return err
		}
		n.operator = NewRemoteOperator(conn, n.Leader())
	}
	return nil
}

func NewNode(ctx context.Context, r *raft.Raft, fsm *Fsm, id string, bootstrapId string, net *network.Network, ipfs *httpapi.HttpApi) (*Node, error) {
	node := &Node{
		raft:        preCommitter{r, fsm.State},
		fsm:         fsm,
		retryTimes:  3,
		ID:          id,
		bootstrapId: bootstrapId,
		operator:    nil,
		mtx:         sync.Mutex{},
		network:     net,
		ctx:         ctx,
		ipfs:        ipfs,
	}
	packer := NewPacker(node.raft, time.Millisecond*300, 100)
	node.packer = packer
	err := node.InitOperator()
	if err != nil {
		return nil, err
	}
	listener, err := gostream.Listen(net.Host(), network.Protocol)
	if err != nil {
		return nil, err
	}
	s1 := grpc.NewServer()

	RegisterRemoteExecuteServer(s1, &FsOpServer{operator: packer, node: node})
	go s1.Serve(listener)
	return node, err
}
