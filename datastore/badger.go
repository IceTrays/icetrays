package datastore

import (
	"errors"
	"github.com/dgraph-io/badger/v3"
)

var (
	dbStateKey     = []byte("state")
	ErrKeyNotFound = errors.New("not found")
)

type BadgerDB struct {
	db *badger.DB
}

func (s *BadgerDB) Delete(bytes []byte) error {
	tx := s.db.NewTransaction(true)
	defer tx.Discard()
	if err := tx.Delete(bytes); err != nil {
		return err
	}
	return tx.Commit()
}

func NewBadgerStore(path string) (*BadgerDB, error) {
	var err error
	store := &BadgerDB{}
	if store.db, err = badger.Open(badger.DefaultOptions(path)); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *BadgerDB) Close() error {
	return s.db.Close()
}

func (s *BadgerDB) Set(key []byte, val []byte) error {
	tx := s.db.NewTransaction(true)
	defer tx.Discard()
	if err := tx.Set(key, val); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *BadgerDB) Get(key []byte) ([]byte, error) {
	tx := s.db.NewTransaction(false)
	defer tx.Discard()
	item, err := tx.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}
	return item.ValueCopy(nil)
}

func (s *BadgerDB) NewTransaction(update bool) Transaction {
	return &Txn{s.db.NewTransaction(update)}
}

func (s *BadgerDB) StoreState(hash string) error {
	return s.Set(dbStateKey, []byte(hash))
}

func (s *BadgerDB) LoadState() (string, error) {
	v, err := s.Get(dbStateKey)
	if err != nil {
		return "", err
	}
	return string(v), err
}

type Txn struct {
	*badger.Txn
}

func (t *Txn) Get(key []byte) ([]byte, error) {
	item, err := t.Txn.Get(key)
	if err != nil {
		return nil, err
	}
	return item.ValueCopy(nil)
}

type Transaction interface {
	kv
	Discard()
	Commit() error
}

type kv interface {
	Delete([]byte) error
	Set(key []byte, val []byte) error
	Get(key []byte) ([]byte, error)
}

type DataBase interface {
	kv
	NewTransaction(update bool) Transaction
}

type StateDB interface {
	StoreState(hash string) error
	LoadState() (string, error)
}
