package datastore

import (
	"encoding/binary"
	"github.com/ipfs/go-cid"
	"sync"
)

var crustOrderPrefix = []byte("crust")

type OrderDB struct {
	db   DataBase
	lock sync.Mutex
}

func (o *OrderDB) AddOrder(fileCid cid.Cid) error {
	o.lock.Lock()
	defer o.lock.Unlock()
	tx := o.db.NewTransaction(true)
	defer tx.Discard()
	key := o.key(fileCid)
	value, err := tx.Get(key)
	if err != nil {
		if err == ErrKeyNotFound {
			value = make([]byte, 8)
		} else {
			return err
		}
	}
	count := binary.LittleEndian.Uint64(value)
	count += 1
	binary.LittleEndian.PutUint64(value, count)
	err = tx.Set(key, value)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (o *OrderDB) DeleteCid(fileCid cid.Cid) (finalCount uint64, err error) {
	o.lock.Lock()
	defer o.lock.Unlock()
	tx := o.db.NewTransaction(true)
	defer tx.Discard()
	key := o.key(fileCid)
	value, err := tx.Get(key)
	if err != nil {
		return 0, err
	}
	finalCount = binary.LittleEndian.Uint64(value)
	finalCount -= 1
	if finalCount == 0 {
		err = tx.Delete(key)
	} else {
		binary.LittleEndian.PutUint64(value, finalCount)
		err = tx.Set(key, value)
	}
	if err != nil {
		return finalCount, err
	}
	return finalCount, tx.Commit()
}

func (o *OrderDB) OrderList() ([]cid.Cid, error) {
	o.lock.Lock()
	defer o.lock.Unlock()
	keys, err := o.db.Keys(crustOrderPrefix)
	if err != nil {
		return nil, err
	}
	res := make([]cid.Cid, len(keys))
	for index, key := range keys {
		res[index], err = cid.Decode(string(key[len(crustOrderPrefix):]))
		if err != nil {
			panic("invalid cid in db")
		}
	}
	return res, nil
}

func (o *OrderDB) key(fileCid cid.Cid) []byte {
	ks := fileCid.String()
	key := make([]byte, len(ks)+5)
	copy(key, crustOrderPrefix)
	copy(key[3:], []byte(ks))
	return key
}
