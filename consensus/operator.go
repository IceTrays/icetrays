package consensus

import (
	"context"
	"fmt"
	"github.com/icetrays/icetrays/consensus/pb"
	"google.golang.org/grpc"
	"time"
)

const defaultTimeout = time.Second * 5

type Operator interface {
	Cp(ctx context.Context, dir, path string, nodeData []byte) error
	Mv(ctx context.Context, dir, path string) error
	Rm(ctx context.Context, path string) error
	MkDir(ctx context.Context, path string) error
	Address() string
	AddPeer(ctx context.Context, id string) error
	PinFile(ctx context.Context, pinNode, cid string) error
	UnPinFile(ctx context.Context, cid string) error
}

type Sender interface {
	Send(*pb.Instruction) error
	AddVoter(nodeId string) error
}

type LocalOperator struct {
	sender Sender
	addr   string
}

func (l *LocalOperator) Cp(ctx context.Context, dir, path string, nodeData []byte) error {
	return l.operation(pb.Instruction_CP, nil, dir, path)
}

func (l *LocalOperator) Mv(ctx context.Context, dir, path string) error {
	return l.operation(pb.Instruction_MV, nil, dir, path)
}

func (l *LocalOperator) Rm(ctx context.Context, path string) error {
	return l.operation(pb.Instruction_RM, nil, path)
}

func (l *LocalOperator) MkDir(ctx context.Context, path string) error {
	return l.operation(pb.Instruction_MKDIR, nil, path)
}

func (l *LocalOperator) Address() string {
	return l.addr
}

func (l *LocalOperator) AddPeer(ctx context.Context, id string) error {
	return l.sender.AddVoter(id)
}

func (l *LocalOperator) PinFile(ctx context.Context, pinNode, cid string) error {
	return l.operation(pb.Instruction_Pin, nil, pinNode, cid)
}

func (l *LocalOperator) UnPinFile(ctx context.Context, cid string) error {
	return l.operation(pb.Instruction_UnPin, nil, cid)
}

func NewLocalOperator(r Sender, address string) *LocalOperator {
	return &LocalOperator{
		sender: r,
		addr:   address,
	}
}

func (l *LocalOperator) operation(code pb.Instruction_Code, nodeData []byte, params ...string) error {
	op := &pb.Instruction{
		Code:   code,
		Params: params,
		Node:   nodeData,
	}
	return l.sender.Send(op)
}

type RemoteOperator struct {
	client RemoteExecuteClient
	addr   string
}

func (r *RemoteOperator) Cp(ctx context.Context, dir, path string, nodeData []byte) error {
	_, err := r.client.Execute(ctx, &pb.Instruction{
		Code:   pb.Instruction_CP,
		Params: []string{dir, path},
		Node:   nodeData,
	})
	return err
}

func (r *RemoteOperator) Mv(ctx context.Context, dir, path string) error {
	_, err := r.client.Execute(ctx, &pb.Instruction{
		Code:   pb.Instruction_MV,
		Params: []string{dir, path},
	})
	return err
}

func (r *RemoteOperator) Rm(ctx context.Context, path string) error {
	_, err := r.client.Execute(ctx, &pb.Instruction{
		Code:   pb.Instruction_RM,
		Params: []string{path},
	})
	return err
}

func (r *RemoteOperator) MkDir(ctx context.Context, path string) error {
	_, err := r.client.Execute(ctx, &pb.Instruction{
		Code:   pb.Instruction_MKDIR,
		Params: []string{path},
	})
	return err
}

func (r *RemoteOperator) Address() string {
	return r.addr
}

func (r *RemoteOperator) AddPeer(ctx context.Context, id string) error {
	fmt.Println(id + " add peer")
	_, err := r.client.AddPeer(ctx, &pb.Node{Id: id})
	return err
}

func (r *RemoteOperator) PinFile(ctx context.Context, pinNode, cid string) error {
	_, err := r.client.Execute(ctx, &pb.Instruction{
		Code:   pb.Instruction_Pin,
		Params: []string{pinNode, cid},
	})
	return err
}

func (r *RemoteOperator) UnPinFile(ctx context.Context, cid string) error {
	_, err := r.client.Execute(ctx, &pb.Instruction{
		Code:   pb.Instruction_UnPin,
		Params: []string{cid},
	})
	return err
}

func NewRemoteOperator(conn grpc.ClientConnInterface, addr string) *RemoteOperator {
	return &RemoteOperator{
		client: NewRemoteExecuteClient(conn),
		addr:   addr,
	}
}

type FsOpServer struct {
	operator Sender
	node     *Node
}

func (f *FsOpServer) Execute(ctx context.Context, instruction *pb.Instruction) (*pb.Empty, error) {
	err := f.operator.Send(instruction)
	return &pb.Empty{}, err
}

func (f *FsOpServer) AddPeer(ctx context.Context, node *pb.Node) (*pb.Empty, error) {
	err := f.node.SwitchOperator()
	if err != nil {
		return &pb.Empty{}, err
	}
	return &pb.Empty{}, f.node.operator.AddPeer(ctx, node.Id)
}

func (f *FsOpServer) mustEmbedUnimplementedRemoteExecuteServer() {

}
