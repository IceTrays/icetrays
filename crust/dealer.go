package crust

import (
	"errors"
	"github.com/ipfs/go-cid"
	"sync"
)

type OrderClient interface {
	PlaceStorageOrder(fileCid cid.Cid, fileSize uint64, tip uint64) error
	GetFileInfo(fileCid cid.Cid) (*FileInfo, error)
}

type Dealer struct {
	client OrderClient
	traced map[string]bool
	mtx sync.Mutex
}

func(dealer *Dealer) AddOrder(cid cid.Cid) error {
	dealer.mtx.Lock()
	defer dealer.mtx.Unlock()
	if dealer.traced[cid.String()] {
		return nil
	}
	if info, err := dealer.client.GetFileInfo(cid); err == nil {
		// found exist same file
		order := Order{
			cid:    cid,
			info:   *info,
			status: OrderStatusAccepted,
		}
		dealer.traced[cid.String()] = true
	} else {
		if errors.Is(err, ErrCidNotFound) {

		}
	}

	return nil
}

func(dealer *Dealer) RemoveOrder(cid cid.Cid) error {
	return nil
}


