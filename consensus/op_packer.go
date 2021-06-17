package consensus

import (
	"errors"
	"github.com/icetrays/icetrays/consensus/pb"
	"sync"
	"time"
)

var ErrShutdown = errors.New("packer shutdown")

type Caller interface {
	Call([]*pb.Instruction) []error
	AddPeer(nodeId string) error
}

type FileOpRequest struct {
	done chan *FileOpRequest
	ins  *pb.Instruction
	err  error
}

type OpPacker struct {
	chanPool sync.Pool
	mtx      sync.Mutex
	cache    []*FileOpRequest
	quit     chan bool
	request  chan *FileOpRequest
	shutdown bool
	caller   Caller
	node     Node
}

func (packer *OpPacker) AddVoter(nodeId string) error {
	return packer.caller.AddPeer(nodeId)
}

func (packer *OpPacker) Send(ins *pb.Instruction) error {
	call, err := packer.send(ins)
	if err != nil {
		return err
	}
	defer packer.chanPool.Put(call.done)
	call = <-call.done
	return call.err
}

func (packer *OpPacker) send(ins *pb.Instruction) (*FileOpRequest, error) {
	packer.mtx.Lock()
	if packer.shutdown == true {
		return nil, ErrShutdown
	}
	packer.mtx.Unlock()
	done := packer.chanPool.Get().(chan *FileOpRequest)
	call := &FileOpRequest{
		done: done,
		ins:  ins,
		err:  nil,
	}
	packer.request <- call
	return call, nil
}

func (packer *OpPacker) Backend(commitTimeout time.Duration, max int) {
	timer := time.NewTimer(commitTimeout)
	for {
		select {
		case r, ok := <-packer.request:
			if !ok {
				packer.request = nil
				continue
			}
			packer.cache = append(packer.cache, r)
			if len(packer.cache) >= max {
				packer.Commit()
				timer.Reset(commitTimeout)
			}
		case <-timer.C:
			packer.Commit()
			timer.Reset(commitTimeout)
		case <-packer.quit:
			timer.Stop()
			packer.Commit()
			return
		}
	}
}

func (packer *OpPacker) Commit() {
	if len(packer.cache) != 0 {
		inss := make([]*pb.Instruction, len(packer.cache))
		for index, call := range packer.cache {
			inss[index] = call.ins
		}
		errs := packer.caller.Call(inss)
		for index, call := range packer.cache {
			call.err = errs[index]
			call.done <- call
		}
		packer.cache = packer.cache[0:0]
	}
}

func (packer *OpPacker) Stop() {
	packer.mtx.Lock()
	packer.shutdown = true
	packer.mtx.Unlock()
	packer.quit <- true
	close(packer.quit)
	close(packer.request)
}

func NewPacker(executor Caller, commitTimeout time.Duration, max int) *OpPacker {
	packer := OpPacker{
		chanPool: sync.Pool{New: func() interface{} { return make(chan *FileOpRequest, 1) }},
		mtx:      sync.Mutex{},
		cache:    make([]*FileOpRequest, 0),
		quit:     make(chan bool),
		request:  make(chan *FileOpRequest),
		shutdown: false,
		caller:   executor,
	}
	go packer.Backend(commitTimeout, max)
	return &packer
}
