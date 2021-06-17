package consensus

import (
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/icetrays/icetrays/consensus/pb"
	"github.com/icetrays/icetrays/consensus/state"
	"time"
)

type Executor interface {
	Execute(ins *pb.Instruction) error
	StartPreExecute() string
	EndPreExecute()
	FailedPreExecute(s string, retry bool) error
}

type OnlyOneCanDo interface {
	Lock() state.SnapShot
	Execute(ins *pb.Instruction) error
	UnLock() state.SnapShot
	SnapShot() state.SnapShot
	RollBack(shot state.SnapShot) error
}

//type IpfsExecutor interface {
//	PinCidFile(ctx context.Context, cid cid.Cid) error
//}

type preCommitter struct {
	*raft.Raft
	preExecutor OnlyOneCanDo
	//ipfsExecutor IpfsExecutor
}

// Call leader先做 做好了发送给flower
func (r preCommitter) Call(instructions []*pb.Instruction) []error {
	errs := make([]error, len(instructions))
	copyIns := make([]*pb.Instruction, 0, len(instructions))
	snapshot := r.preExecutor.Lock()
	for index, ins := range instructions {
		errs[index] = r.preExecutor.Execute(ins)
		if errs[index] == nil {
			copyIns = append(copyIns, ins)
		}
	}
	after := r.preExecutor.UnLock()
	for r.preExecutor.RollBack(snapshot) != nil {
		time.Sleep(time.Second)
	}
	inss := pb.Instructions{
		Instruction: copyIns,
		Ctx: &pb.Ctx{
			Pre:  snapshot.Root,
			Next: after.Root,
		},
	}
	bs, err := proto.Marshal(&inss)
	if err != nil {
		// todo log
		return CopyError(err, len(instructions))
	}
	future := r.Apply(bs, defaultTimeout)
	if future.Error() != nil {
		return CopyError(err, len(instructions))
	}
	switch res := future.Response().(type) {
	case error:
		return CopyError(res, len(instructions))
	default:
		return errs
	}
}

func (r preCommitter) AddPeer(nodeId string) error {
	addFuture := r.AddVoter(raft.ServerID(nodeId), raft.ServerAddress(nodeId), 0, 0)
	return addFuture.Error()
}

func CopyError(err error, num int) []error {
	errs := make([]error, num)
	for i := 0; i < num; i++ {
		errs[i] = err
	}
	return errs
}
