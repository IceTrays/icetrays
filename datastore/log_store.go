package datastore

import (
	"encoding/binary"
	"github.com/hashicorp/raft"
	"github.com/icetrays/icetrays/datastore/pb"
	"github.com/ipfs/go-log/v2"
)

var (
	dbLogsFirstIndex = []byte("l_first")
	dbLogsLastIndex  = []byte("l_last")
)

var logger = log.Logger("db")

type LogDB struct {
	db DataBase
}

func (l LogDB) FirstIndex() (uint64, error) {
	if v, err := l.db.Get(dbLogsFirstIndex); err == nil {
		return binary.LittleEndian.Uint64(v), nil

	} else {
		if err == ErrKeyNotFound {
			return 0, nil
		}
		return 0, err
	}
}

func (l LogDB) LastIndex() (uint64, error) {
	if v, err := l.db.Get(dbLogsLastIndex); err == nil {
		return binary.LittleEndian.Uint64(v), nil

	} else {
		if err == ErrKeyNotFound {
			return 0, nil
		}
		return 0, err
	}
}

func (l LogDB) GetLog(index uint64, log *raft.Log) error {
	logger.Infof("get log : %d", index)
	if bs, err := l.db.Get(l.uint64Key(index)); err == nil {
		return pb.DecodeLog(bs, log)
	} else {
		if err == ErrKeyNotFound {
			return ErrKeyNotFound
		}
		return err
	}
}

func (l LogDB) StoreLog(log *raft.Log) error {
	tx := l.db.NewTransaction(true)
	defer tx.Discard()
	bs, err := pb.EncodeLog(log)
	if err != nil {
		return err
	}
	if err := tx.Set(l.uint64Key(log.Index), bs); err != nil {
		return err
	}
	tmp := make([]byte, 8, 8)
	binary.LittleEndian.PutUint64(tmp, log.Index)
	if err := tx.Set(dbLogsLastIndex, tmp); err != nil {
		return err
	}
	return tx.Commit()
}

func (l LogDB) StoreLogs(logs []*raft.Log) error {
	tx := l.db.NewTransaction(true)
	defer tx.Discard()
	key := make([]byte, 9)
	key[0] = l.prefix()
	var max uint64
	for _, logObj := range logs {
		binary.LittleEndian.PutUint64(key[1:], logObj.Index)

		bs, err := pb.EncodeLog(logObj)
		if err != nil {
			return err
		}
		tmp := make([]byte, 9)
		copy(tmp, key)
		if err := tx.Set(tmp, bs); err != nil {
			return err
		}
		if logObj.Index > max {
			max = logObj.Index
		}
	}
	value := make([]byte, 8, 8)
	binary.LittleEndian.PutUint64(value, max)
	if err := tx.Set(dbLogsLastIndex, value); err != nil {
		return err
	}
	return tx.Commit()
}

func (l LogDB) DeleteRange(min, max uint64) error {
	logger.Infof("DeleteRange: %d-%d", min, max)
	tx := l.db.NewTransaction(true)
	defer tx.Discard()
	key := make([]byte, 9)
	key[0] = l.prefix()
	lowIndex, err := l.FirstIndex()
	if err != nil {
		return err
	}
	highIndex, err := l.LastIndex()
	if err != nil {
		return err
	}
	for i := min; i <= max; i++ {
		binary.LittleEndian.PutUint64(key[1:], i)
		tmp := make([]byte, 9)
		copy(tmp, key)
		if err := tx.Delete(tmp); err != nil && err != ErrKeyNotFound {
			return err
		}
	}
	if min <= lowIndex {
		lowIndex = max + 1
	}
	if max >= highIndex {
		highIndex = min - 1
	}
	if lowIndex > highIndex {
		highIndex = 0
		lowIndex = 0
	}
	key = make([]byte, 8, 8)
	binary.LittleEndian.PutUint64(key, lowIndex)
	tmp := make([]byte, 9)
	copy(tmp, key)
	if err := tx.Set(dbLogsFirstIndex, tmp); err != nil {
		return err
	}
	binary.LittleEndian.PutUint64(key, highIndex)
	tmp2 := make([]byte, 9)
	copy(tmp2, key)
	if err := tx.Set(dbLogsLastIndex, tmp2); err != nil {
		return err
	}
	return tx.Commit()
}

func (l LogDB) prefix() byte {
	return 'l'
}

func (l LogDB) uint64Key(index uint64) []byte {
	keyInDB := make([]byte, 9)
	keyInDB[0] = l.prefix()
	binary.LittleEndian.PutUint64(keyInDB[1:], index)
	return keyInDB
}

func NewLogDB(db *BadgerDB) *LogDB {
	return &LogDB{db}
}
