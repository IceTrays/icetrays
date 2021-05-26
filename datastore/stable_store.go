package datastore

import (
	"encoding/binary"
)

type StableDB struct {
	db DataBase
}

func (s *StableDB) Set(key []byte, val []byte) error {
	keyInDB := []byte{s.prefix()}
	keyInDB = append(keyInDB, key...)
	return s.db.Set(keyInDB, val)
}

func (s *StableDB) Get(key []byte) ([]byte, error) {
	keyInDB := []byte{s.prefix()}
	keyInDB = append(keyInDB, key...)
	return s.db.Get(keyInDB)
}

func (s *StableDB) SetUint64(key []byte, val uint64) error {
	tmp := make([]byte, 8, 8)
	binary.LittleEndian.PutUint64(tmp, val)
	return s.Set(key, tmp)
}

func (s *StableDB) GetUint64(key []byte) (uint64, error) {
	if tmp, err := s.Get(key); err != nil {
		return 0, err
	} else {
		return binary.LittleEndian.Uint64(tmp), nil
	}
}

func (s *StableDB) prefix() byte {
	return byte('s')
}

func NewStableDB(db *BadgerDB) *StableDB {
	return &StableDB{db}
}
