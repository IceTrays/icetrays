package consensus

import (
	"context"
	"errors"
	"github.com/hashicorp/raft"
	"github.com/icetrays/icetrays/network"
	"github.com/icetrays/icetrays/types"
	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/go-mfs"
	gostream "github.com/libp2p/go-libp2p-gostream"
	"google.golang.org/grpc"
	"strings"
	"sync"
	"time"
)

type Node struct {
	raft       preCommitter
	fsm        *Fsm
	retryTimes int
	ID         string
	operator   Operator
	mtx        sync.Mutex
	network    *network.Network
	ctx        context.Context
	ipfs       *httpapi.HttpApi
	packer     Sender
}

func (n *Node) Op(ctx context.Context, code types.InstructionCode, params ...string) error {
	if n.fsm.Inconsistent() {
		return errors.New("inconsistent state")
	}

	err := n.TrySwitchOperator()
	if err != nil {
		return err
	}
	switch code {
	case types.InstructionCP:
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
	case types.InstructionMV:
		return n.operator.Mv(ctx, params[0], params[1])
	case types.InstructionRM:
		return n.operator.Rm(ctx, params[0])
	case types.InstructionMKDIR:
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

func (n *Node) Operator() string {
	return n.operator.Address()
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

func NewNode(ctx context.Context, r *raft.Raft, fsm *Fsm, id string, net *network.Network, ipfs *httpapi.HttpApi) (*Node, error) {
	node := &Node{
		raft:       preCommitter{r, fsm.State},
		fsm:        fsm,
		retryTimes: 3,
		ID:         id,
		operator:   nil,
		mtx:        sync.Mutex{},
		network:    net,
		ctx:        ctx,
		ipfs:       ipfs,
	}
	err := node.SwitchOperator()
	listener, err := gostream.Listen(net.Host(), network.Protocol)
	if err != nil {
		return nil, err
	}
	packer := NewPacker(node.raft, time.Millisecond*300, 100)
	node.packer = packer
	s1 := grpc.NewServer()

	RegisterRemoteExecuteServer(s1, FsOpServer{operator: packer})
	go s1.Serve(listener)
	return node, err
}
