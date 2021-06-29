package consensus

import (
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
	"time"
)

type Executor interface {
	Execute(ins *Instruction) error
	StartPreExecute() string
	EndPreExecute()
	FailedPreExecute(s string, retry bool) error
}

type OnlyOneCanDo interface {
	Lock() SnapShot
	Execute(ins *Instruction) error
	UnLock() SnapShot
	SnapShot() SnapShot
	RollBack(shot SnapShot) error
}

type preCommitter struct {
	*raft.Raft
	preExecutor OnlyOneCanDo
}

func (r preCommitter) Call(instructions []*Instruction) []error {
	errs := make([]error, len(instructions))
	copyIns := make([]*Instruction, 0, len(instructions))
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
	inss := Instructions{
		Instruction: copyIns,
		Ctx: &Ctx{
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

func CopyError(err error, num int) []error {
	errs := make([]error, num)
	for i := 0; i < num; i++ {
		errs[i] = err
	}
	return errs
}
