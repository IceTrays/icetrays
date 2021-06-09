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
	BlockNumber() (types.BlockNumber, error)
}

type Dealer struct {
	client         OrderClient
	traced         map[string]*Order
	mtx            sync.Mutex
	group          sync.WaitGroup
	quit           chan bool
	tickerDuration time.Duration
	currentHeight  types.BlockNumber
	renewHeight    types.BlockNumber
}

func (dealer *Dealer) AddOrder(cid cid.Cid, fileSize uint64) {
	newOrder := &Order{
		cid:      cid,
		fileSize: fileSize,
		status:   OrderStatusStart,
	}
	dealer.mtx.Lock()
	dealer.traced[cid.String()] = newOrder
	dealer.mtx.Unlock()
	dealer.group.Add(1)
	go func() {
		timer := time.NewTimer(time.Second / 2)
		defer func() {
			timer.Stop()
			dealer.group.Done()
		}()
		for {
			select {
			case <-timer.C:
				fmt.Println(newOrder.Status())
				switch newOrder.Status() {
				case OrderStatusStart:
					dealer.solveOrderStatusStart(newOrder)
				case OrderStatusWaiting:
					dealer.solveOrderStatusWaiting(newOrder)
				case OrderStatusAccepted:
					dealer.solveOrderStatusAccepted(newOrder)
				case OrderStatusRenew:
				case OrderStatusRetry:
					dealer.solveOrderStatusRenew(newOrder)
				case OrderStatusError:
					// TODO do nothing
					fmt.Println(newOrder.Error())
				case OrderStatusDiscard:
					return
				}
			case <-dealer.quit:
				return
			}
			_ = timer.Reset(dealer.tickerDuration)
		}
	}()
}

func (dealer *Dealer) RemoveOrder(cid cid.Cid) error {
	dealer.mtx.Lock()
	order := dealer.traced[cid.String()]
	dealer.mtx.Unlock()
	if order == nil {
		return ErrCidNotFound
	}
	order.SetStatus(OrderStatusDiscard)
	return nil
}

func (dealer *Dealer) Stop() {
	close(dealer.quit)
	dealer.group.Wait()
}

func (dealer *Dealer) Start() {
	dealer.group.Add(1)
	go func() {
		ticker := time.NewTicker(time.Second * 5)
		defer func() {
			ticker.Stop()
			dealer.group.Done()
		}()
		for {
			select {
			case <-ticker.C:
				height, err := dealer.client.BlockNumber()
				if err != nil {
					fmt.Println(err)
				}
				// todo
				dealer.currentHeight = height
			case <-dealer.quit:
				return
			}
		}
	}()
}

func (dealer *Dealer) solveOrderStatusStart(order *Order) {
	info, err := dealer.client.GetFileInfo(order.Cid())
	if err == nil {
		order.SetFileInfo(info)
	}
	switch {
	case err == nil && info.ExpiredOn > dealer.currentHeight:
		order.SetStatus(OrderStatusAccepted)
	case err == nil && info.ExpiredOn == 0:
		order.WaitingAt(dealer.currentHeight)
	case err != nil && !errors.Is(err, ErrCidNotFound):
		order.SetStatus(OrderStatusRetry)
		if order.ErrorFound(err) >= 3 {
			order.SetStatus(OrderStatusError)
		}
	default:
		err2 := dealer.client.PlaceStorageOrder(order.Cid(), order.FileSize(), 0)
		if err2 != nil {
			order.SetStatus(OrderStatusRetry)
			if order.ErrorFound(err2) >= 3 {
				order.SetStatus(OrderStatusError)
			}
		} else {
			order.WaitingAt(dealer.currentHeight)
		}
	}
}

func (dealer *Dealer) solveOrderStatusWaiting(order *Order) {
	info, err := dealer.client.GetFileInfo(order.Cid())
	if err != nil {
		if errors.Is(err, ErrCidNotFound) {
			panic(fmt.Sprintf("cid order: %s not found", order.Cid().String()))
		}
		return
	}
	order.SetFileInfo(info)
	if info.CalculatedAt >= order.Height() {
		if info.ExpiredOn-dealer.currentHeight < dealer.renewHeight {
			order.SetStatus(OrderStatusRenew)
			return
		}
		order.SetStatus(OrderStatusAccepted)
	}
}

func (dealer *Dealer) solveOrderStatusAccepted(order *Order) {
	info, err := dealer.client.GetFileInfo(order.Cid())
	if err != nil {
		if errors.Is(err, ErrCidNotFound) {
			panic(fmt.Sprintf("cid order: %s not found", order.Cid().String()))
		}
		// TODO log
		return
	}
	order.SetFileInfo(info)
	if info.ExpiredOn-dealer.currentHeight < dealer.renewHeight {
		order.SetStatus(OrderStatusRenew)
	}
}

func (dealer *Dealer) solveOrderStatusRenew(order *Order) {
	err := dealer.client.PlaceStorageOrder(order.cid, order.fileSize, 0)
	if err != nil {
		order.SetStatus(OrderStatusRetry)
		if order.ErrorFound(err) >= 3 {
			order.SetStatus(OrderStatusError)
		}
	} else {
		order.WaitingAt(dealer.renewHeight)
	}
}

func NewDealer(client OrderClient, tickerDuration time.Duration, renewHeight types.BlockNumber) (*Dealer, error) {
	dealer := Dealer{
		client:         client,
		traced:         make(map[string]*Order),
		mtx:            sync.Mutex{},
		group:          sync.WaitGroup{},
		quit:           make(chan bool),
		tickerDuration: tickerDuration,
		renewHeight:    renewHeight,
	}
	height, err := client.BlockNumber()
	if err != nil {
		return nil, err
	}
	dealer.currentHeight = height
	return &dealer, nil
}
