package state

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	badger "github.com/ipfs/go-ds-badger"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"strings"
	"sync"
	"time"
)

const (
	// PinCid /PinCid/cid
	PinCid = "pinCid"
	// PinPeer /PinPeer/cid/peer
	PinPeer = "pinPeer"
	// PinTaskStatus /PinTaskStatus/cid
	PinTaskStatus = "PinTaskStatus"

	MaxRetryCount    = 5
	ScheduleInterval = 300 * time.Second
)

type ss int

const (
	PinUnFinished ss = iota
	PinFinished
	PinFail
)

type PinTask struct {
	ctx    context.Context
	c      cid.Cid
	retry  int
	done   func(err error)
	close  func()
	status ss
}

type PinService struct {
	iface.PinAPI
	mtx        sync.Mutex
	processing map[string]*PinTask
	ipfsDb     *badger.Datastore
}

func NewPinTask(ctx context.Context, c cid.Cid, db *badger.Datastore) (*PinTask, error) {
	sk := ds.KeyWithNamespaces([]string{PinTaskStatus, c.String()})
	done := func(err error) {
		v, de := db.Get(sk)
		if de != nil {
			fmt.Printf("illegal key: %s\n", c.String())
			return
		}
		task := &PinTask{}
		je := json.Unmarshal(v, task)
		if je != nil {
			fmt.Printf("illegal value: %s\n", c.String())
			return
		}
		var r []byte
		if task.retry >= MaxRetryCount {
			fmt.Printf("key:%s reached max number of retries\n", c.String())
			task.status = PinFail
		} else {
			if err != nil {
				fmt.Printf("pin task finished success, cid:%s\n", c.String())
				task.status = PinFinished
			} else {
				task.retry = task.retry + 1
			}
		}
		r, _ = json.Marshal(task)
		_ = db.Put(sk, r)
	}
	ctx, cs := context.WithCancel(ctx)
	task := &PinTask{
		ctx:    ctx,
		c:      c,
		retry:  0,
		done:   done,
		close:  cs,
		status: PinUnFinished,
	}
	r, _ := json.Marshal(task)
	err := db.Put(sk, r)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (p *PinService) Init(ctx context.Context) {
	checkFunc := func() {
		results, err := p.ipfsDb.Query(query.Query{Prefix: PinTaskStatus})
		if err != nil {
			panic(err)
		}
		for r := range results.Next() {
			// get name list
			c := strings.Split(r.Key, "/")[1:][1]
			if p.exist(c) {
				continue
			}
			task := &PinTask{}
			err := json.Unmarshal(r.Value, task)
			if err != nil {
				fmt.Printf("illegal db value:%s\n", r.Key)
				continue
			}
			err = p.restoreTask(ctx, c, task)
			if err != nil {
				fmt.Printf("key:%s restoreTask fail, err:%+v\n", r.Key, err)
			}
		}

	}
	checkFunc()
	tc := time.NewTicker(ScheduleInterval)
	select {
	case <-ctx.Done():
		return
	case <-tc.C:
		checkFunc()
	}
}

func (p *PinService) PinCid(ctx context.Context, c cid.Cid) error {
	task, err := NewPinTask(ctx, c, p.ipfsDb)
	if err != nil {
		return err
	}
	p.put(c.String(), task)
	go func() {
		err := p.Add(task.ctx, path.IpfsPath(c))
		task.done(err)
	}()
	return nil
}

func (p *PinService) UnPin(peerId, cs string) error {
	results, err := p.ipfsDb.Query(query.Query{Prefix: strings.Join([]string{PinPeer, peerId}, "/")})
	if err != nil {
		panic(err)
	}
	for r := range results.Next() {
		_ = p.ipfsDb.Delete(ds.NewKey(r.Key))
	}
	p.delete(cs)
	return nil
}

func (p *PinService) exist(key string) bool {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	_, ok := p.processing[key]
	return ok
}

func (p *PinService) put(key string, task *PinTask) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if _, ok := p.processing[key]; !ok {
		fmt.Printf("add pin task success, cid:%s, retry counts:%d\n", key, task.retry)
		p.processing[key] = task
	}
}

// 1.close context
// 2.delete cache
// 3.delete db value
func (p *PinService) delete(key string) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if v, ok := p.processing[key]; ok {
		v.close()
		delete(p.processing, key)
		_ = p.ipfsDb.Delete(ds.KeyWithNamespaces([]string{PinTaskStatus, key}))
	}
}

func (p *PinService) restoreTask(ctx context.Context, cs string, task *PinTask) error {
	c, err := cid.Decode(cs)
	if err != nil {
		return err
	}
	p.put(cs, task)
	go func() {
		err := p.Add(ctx, path.IpfsPath(c))
		task.done(err)
	}()
	return nil
}
