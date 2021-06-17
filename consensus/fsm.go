package consensus

import (
	"context"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/icetrays/icetrays/consensus/pb"
	"github.com/icetrays/icetrays/consensus/state"
	"github.com/icetrays/icetrays/datastore"
	badger "github.com/ipfs/go-ds-badger"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/go-log/v2"
	"io"
)

var ErrInconsistent = errors.New("inconsistent")

var logger = log.Logger("fsm")

func init() {
	log.SetLogLevel("fsm", "debug")
}

type Fsm struct {
	client       *httpapi.HttpApi
	State        *state.FileTreeState
	ctx          context.Context
	inconsistent bool
}

func NewFsm(store *datastore.BadgerDB, api *httpapi.HttpApi, id string, d *badger.Datastore) (*Fsm, error) {
	_state, err := state.NewFileTreeState(store, id, api.Dag(), api.Pin(), api.Unixfs(), d)
	if err != nil {
		return nil, err
	}
	return &Fsm{
		client:       api,
		State:        _state,
		ctx:          context.Background(),
		inconsistent: false,
	}, nil
}

func (f *Fsm) Apply(log *raft.Log) interface{} {
	if log.Type != raft.LogCommand {
		return nil
	}
	var err error
	index := f.State.Index()
	if log.Index < index {
		f.inconsistent = true
		return ErrInconsistent
	} else if log.Index == index {
		f.inconsistent = false
		return ErrInconsistent
	}
	inss := &pb.Instructions{}
	if err = proto.Unmarshal(log.Data, inss); err != nil {
		return err
	}
	commitFunction := func(insList []*pb.Instruction) error {
		for _, ins := range insList {
			if err := f.State.Execute(ins); err != nil {
				return err
			}
		}
		return nil
	}
	snapshot := f.State.Lock()
	var leader bool
	for {
		var err error
		// todo  ctx ?
		if f.State.MustGetRoot() == inss.Ctx.Next {
			leader = true
			fmt.Println("i am leader.")
		} else {
			err = commitFunction(inss.Instruction)
			if err != nil {
				f.State.MustRollBack(snapshot)
				continue
			}
		}

		f.State.SetIndex(log.Index)
		for {
			if err := f.State.Flush(); err == nil {
				break
			}
		}
		break
	}
	after := f.State.UnLock()
	if (snapshot.Root != inss.Ctx.Pre && !leader) || after.Root != inss.Ctx.Next {
		logger.Warnf("inconsistent: want: %s->%s, got %s->%s", inss.Ctx.Pre, inss.Ctx.Next, snapshot.Root, after.Root)
		//_ = f.State.Unmarshal(strings.NewReader(inss.Ctx.Next))
	}
	return nil
}

func (f *Fsm) Snapshot() (raft.FSMSnapshot, error) {
	return &Snapshot{state: f.State}, nil
}

func (f *Fsm) Restore(closer io.ReadCloser) error {
	defer closer.Close()
	return f.State.Unmarshal(closer)
}

func (f *Fsm) StoreConfiguration(index uint64, config raft.Configuration) {
	fmt.Println("receive raft LogConfiguration log...")
	fmt.Printf("config index: %d, config servers: %v\n", index, config.Servers)
}

func (f *Fsm) Inconsistent() bool {
	return f.inconsistent
}

type Snapshot struct {
	state *state.FileTreeState
}

func (s *Snapshot) Persist(sink raft.SnapshotSink) error {
	return s.state.Marshal(sink)
}

func (s Snapshot) Release() {

}
