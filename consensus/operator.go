package consensus

import (
	"context"
	"github.com/gogo/protobuf/proto"
	"github.com/icetrays/icetrays/types"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-path"
	"google.golang.org/grpc"
	"time"
)

const defaultTimeout = time.Second * 5

type Operator interface {
	Cp(ctx context.Context, file cid.Cid, dir path.Path, PinCount uint32, Crust bool, NodeData []byte) error
	Mv(ctx context.Context, from path.Path, to path.Path) error
	Rm(ctx context.Context, dir path.Path) error
	Mkdir(ctx context.Context, dir path.Path) error
	Pin(ctx context.Context, info types.PinInfo) error
	UnPin(ctx context.Context, file cid.Cid) error
	Address() string
}

type Sender interface {
	Send(*Instruction) error
}

type LocalOperator struct {
	sender Sender
	addr   string
}

func (l *LocalOperator) Cp(ctx context.Context, file cid.Cid, dir path.Path, PinCount uint32, Crust bool, NodeData []byte) error {
	params := types.CpParams{
		Path:     dir.String(),
		Cid:      file.String(),
		NodeData: NodeData,
		PinNums:  PinCount,
		Crust:    Crust,
	}
	bs, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	return l.sender.Send(&Instruction{
		Code:   types.InstructionCP,
		Params: bs,
	})
}

func (l *LocalOperator) Mv(ctx context.Context, from path.Path, to path.Path) error {
	params := types.MvParams{
		From: from.String(),
		To:   to.String(),
	}
	bs, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	return l.sender.Send(&Instruction{
		Code:   types.InstructionMV,
		Params: bs,
	})
}

func (l *LocalOperator) Rm(ctx context.Context, dir path.Path) error {
	params := types.RmParams{
		Path: dir.String(),
	}
	bs, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	return l.sender.Send(&Instruction{
		Code:   types.InstructionRM,
		Params: bs,
	})
}

func (l *LocalOperator) Mkdir(ctx context.Context, dir path.Path) error {
	params := types.MkdirParams{
		Path: dir.String(),
	}
	bs, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	return l.sender.Send(&Instruction{
		Code:   types.InstructionMKDIR,
		Params: bs,
	})
}

func (l *LocalOperator) Pin(ctx context.Context, info types.PinInfo) error {
	params := types.PinParams{
		Cid:     info.Cid.String(),
		PinNums: info.PinCount,
		Crust:   info.Crust,
	}
	bs, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	return l.sender.Send(&Instruction{
		Code:   types.InstructionPIN,
		Params: bs,
	})
}

func (l *LocalOperator) UnPin(ctx context.Context, file cid.Cid) error {
	params := types.UnpinParams{
		Cid: file.String(),
	}
	bs, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	return l.sender.Send(&Instruction{
		Code:   types.InstructionUNPIN,
		Params: bs,
	})
}

func (l *LocalOperator) Address() string {
	return l.addr
}

func NewLocalOperator(r Sender, address string) *LocalOperator {
	return &LocalOperator{
		sender: r,
		addr:   address,
	}
}

type RemoteOperator struct {
	client RemoteExecuteClient
	addr   string
}

func (r *RemoteOperator) Cp(ctx context.Context, file cid.Cid, dir path.Path, PinCount uint32, Crust bool, NodeData []byte) error {
	params := types.CpParams{
		Path:     dir.String(),
		Cid:      file.String(),
		NodeData: NodeData,
		PinNums:  PinCount,
		Crust:    Crust,
	}
	bs, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	_, err = r.client.Execute(ctx, &Instruction{
		Code:   types.InstructionCP,
		Params: bs,
	})
	return err
}

func (r *RemoteOperator) Mv(ctx context.Context, from path.Path, to path.Path) error {
	params := types.MvParams{
		From: from.String(),
		To:   to.String(),
	}
	bs, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	_, err = r.client.Execute(ctx, &Instruction{
		Code:   types.InstructionMV,
		Params: bs,
	})
	return err
}

func (r *RemoteOperator) Rm(ctx context.Context, dir path.Path) error {
	params := types.RmParams{
		Path: dir.String(),
	}
	bs, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	_, err = r.client.Execute(ctx, &Instruction{
		Code:   types.InstructionRM,
		Params: bs,
	})
	return err
}

func (r *RemoteOperator) Mkdir(ctx context.Context, dir path.Path) error {
	params := types.MkdirParams{
		Path: dir.String(),
	}
	bs, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	_, err = r.client.Execute(ctx, &Instruction{
		Code:   types.InstructionMKDIR,
		Params: bs,
	})
	return err
}

func (r *RemoteOperator) Pin(ctx context.Context, info types.PinInfo) error {
	params := types.PinParams{
		Cid:     info.Cid.String(),
		PinNums: info.PinCount,
		Crust:   info.Crust,
	}
	bs, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	_, err = r.client.Execute(ctx, &Instruction{
		Code:   types.InstructionPIN,
		Params: bs,
	})
	return err
}

func (r *RemoteOperator) UnPin(ctx context.Context, file cid.Cid) error {
	params := types.UnpinParams{
		Cid: file.String(),
	}
	bs, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	_, err = r.client.Execute(ctx, &Instruction{
		Code:   types.InstructionUNPIN,
		Params: bs,
	})
	return err
}

func (r *RemoteOperator) Address() string {
	return r.addr
}

func NewRemoteOperator(conn grpc.ClientConnInterface, addr string) *RemoteOperator {
	return &RemoteOperator{
		client: NewRemoteExecuteClient(conn),
		addr:   addr,
	}
}

type FsOpServer struct {
	operator Sender
}

func (f FsOpServer) Execute(ctx context.Context, instruction *Instruction) (*Empty, error) {
	err := f.operator.Send(instruction)
	return &Empty{}, err
}

func (f FsOpServer) mustEmbedUnimplementedRemoteExecuteServer() {

}
