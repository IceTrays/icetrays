package crust

import (
	"errors"
	"fmt"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/ipfs/go-cid"
	"sync"
	"time"
)

type OrderClient interface {
	PlaceStorageOrder(fileCid cid.Cid, fileSize uint64, tip uint64) error
	GetFileInfo(fileCid cid.Cid) (*FileInfo, error)
}

type Dealer struct {
	client         OrderClient
	traced         map[string]*Order
	mtx            sync.Mutex
	group          sync.WaitGroup
	quit           chan bool
	tickerDuration time.Duration
	currentHeight  types.BlockNumber
}

func (dealer *Dealer) AddOrder(cid cid.Cid) error {
	//dealer.mtx.Lock()
	//defer dealer.mtx.Unlock()
	//if dealer.traced[cid.String()] != nil {
	//	return nil
	//}
	//if info, err := dealer.client.GetFileInfo(cid); err == nil {
	//	// found exist same file
	//	order := Order{
	//		cid:    cid,
	//		info:   *info,
	//		status: OrderStatusAccepted,
	//	}
	//	dealer.traced[cid.String()] = true
	//} else {
	//	if errors.Is(err, ErrCidNotFound) {
	//
	//	}
	//}

	return nil
}

func (dealer *Dealer) RemoveOrder(cid cid.Cid) error {
	return nil
}

func (dealer *Dealer) solve(order *Order) {
	switch order.status {

	}
}

func (dealer *Dealer) solveOrderStatusStart(order *Order) {
	info, err := dealer.client.GetFileInfo(order.cid)
	switch {
	case err == nil && info.ExpiredOn > dealer.currentHeight:
		order.SetStatus(OrderStatusAccepted)
	case err == nil && info.ExpiredOn == 0:
		order.height = dealer.currentHeight
		order.SetStatus(OrderStatusWaiting)
	case err != nil && !errors.Is(err, ErrCidNotFound):
		order.err = err
		order.SetStatus(OrderStatusRetry)
		order.retryTimes += 1
		if order.retryTimes >= 3 {
			order.SetStatus(OrderStatusError)
		}
	default:
		order.err = dealer.client.PlaceStorageOrder(order.cid, order.fileSize, 0)
		if order.err != nil {
			order.SetStatus(OrderStatusRetry)
			order.retryTimes += 1
			if order.retryTimes >= 3 {
				order.SetStatus(OrderStatusError)
			}
		} else {
			order.height = dealer.currentHeight
			order.SetStatus(OrderStatusWaiting)
		}
	}
}

func (dealer *Dealer) solveOrderStatusWaiting(order *Order) {
	info, err := dealer.client.GetFileInfo(order.cid)
	if err != nil {
		if errors.Is(err, ErrCidNotFound) {
			panic(fmt.Sprintf("cid order: %s not found", order.cid.String()))
		}
		// TODO log
		return
	}
	if info.CalculatedAt > order.height {
		if info.ExpiredOn < dealer.currentHeight {
			order.SetStatus(OrderStatusRenew)
			return
		}
		order.SetStatus(OrderStatusAccepted)
	}
}
