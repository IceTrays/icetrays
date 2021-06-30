package consensus

import (
	"context"
	"github.com/hashicorp/raft"
	"github.com/icetrays/icetrays/network"
	"github.com/icetrays/icetrays/types"
	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/go-path"
	gostream "github.com/libp2p/go-libp2p-gostream"
	"google.golang.org/grpc"
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

func (n *Node) Cp(ctx context.Context, file cid.Cid, dir path.Path, info types.PinInfo) error {
	cctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	ipldNode, err := n.ipfs.Dag().Get(cctx, file)
	if err != nil {
		return err
	}
	if err := n.TrySwitchOperator(); err != nil {
		return err
	}
	return n.operator.Cp(ctx, file, dir, info.PinCount, info.Crust, ipldNode.RawData())
}

func (n *Node) Ls(ctx context.Context, dir path.Path) ([]types.LsFileInfo, error) {
	// TODO
	return nil, nil
}

func (n *Node) Mv(ctx context.Context, from path.Path, to path.Path) error {
	if err := n.TrySwitchOperator(); err != nil {
		return err
	}
	return n.operator.Mv(ctx, from, to)
}

func (n *Node) Rm(ctx context.Context, dir path.Path) error {
	if err := n.TrySwitchOperator(); err != nil {
		return err
	}
	return n.operator.Rm(ctx, dir)
}

func (n *Node) Mkdir(ctx context.Context, dir path.Path) error {
	if err := n.TrySwitchOperator(); err != nil {
		return err
	}
	return n.operator.Mkdir(ctx, dir)
}

func (n *Node) Pin(ctx context.Context, info types.PinInfo) error {
	if err := n.TrySwitchOperator(); err != nil {
		return err
	}
	return n.operator.Pin(ctx, info)
}

func (n *Node) UnPin(ctx context.Context, file cid.Cid) error {
	if err := n.TrySwitchOperator(); err != nil {
		return err
	}
	return n.operator.UnPin(ctx, file)
}

func (n *Node) Stat(ctx context.Context, cid cid.Cid) (types.LsFileInfo, error) {
	panic("implement me")
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
	// todo error handle
	go s1.Serve(listener)
	return node, err
}
