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
	PlaceStorageOrder(fileCid cid.Cid, fileSize uint64, tip uint64, needCalculate bool) error
	GetFileInfo(fileCid cid.Cid) (*FileInfo, error)
	BlockNumber() (types.BlockNumber, error)
}

type OrderStore interface {
	AddOrder(fileCid cid.Cid) error
	DeleteCid(fileCid cid.Cid) (count uint64, err error)
	OrderList() ([]cid.Cid, error)
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
	store          OrderStore
}

func (dealer *Dealer) AddOrder(cid cid.Cid, fileSize uint64) error {
	newOrder := &Order{
		cid:      cid,
		fileSize: fileSize,
		status:   OrderStatusStart,
	}
	err := dealer.store.AddOrder(cid)
	if err != nil {
		return err
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
				fmt.Printf("%v: %+v, info: %+v\n", newOrder.Status(), newOrder, newOrder.info)
				switch newOrder.Status() {
				case OrderStatusStart:
					dealer.solveOrderStatusStart(newOrder)
				case OrderStatusWaiting:
					dealer.solveOrderStatusWaiting(newOrder)
				case OrderStatusAccepted:
					dealer.solveOrderStatusAccepted(newOrder)
				case OrderStatusRenew:
					fallthrough
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
	return nil
}

func (dealer *Dealer) RemoveOrder(cid cid.Cid) error {
	count, err := dealer.store.DeleteCid(cid)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
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
				dealer.SetCurrentHeight(height)
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
	case err == nil && info.ExpiredOn > dealer.CurrentHeight():
		order.SetStatus(OrderStatusAccepted)
	case err == nil && info.ExpiredOn == 0:
		order.WaitingAt(dealer.CurrentHeight())
	case err != nil && !errors.Is(err, ErrCidNotFound):
		order.SetStatus(OrderStatusRetry)
		if order.ErrorFound(err) >= 3 {
			order.SetStatus(OrderStatusError)
		}
	default:
		var needCalculate bool
		if info != nil {
			needCalculate = dealer.CurrentHeight() > info.ExpiredOn
		}
		err2 := dealer.client.PlaceStorageOrder(order.Cid(), order.FileSize(), 0, needCalculate)
		if err2 != nil {
			order.SetStatus(OrderStatusRetry)
			if order.ErrorFound(err2) >= 3 {
				order.SetStatus(OrderStatusError)
			}
		} else {
			order.WaitingAt(dealer.CurrentHeight())
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
		if info.ExpiredOn-dealer.CurrentHeight() < dealer.renewHeight {
			order.SetStatus(OrderStatusRenew)
			return
		}
		if info.ExpiredOn != 0 {
			order.SetStatus(OrderStatusAccepted)
		}
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
	if info.ExpiredOn-dealer.CurrentHeight() < dealer.renewHeight {
		order.SetStatus(OrderStatusRenew)
	}
}

func (dealer *Dealer) solveOrderStatusRenew(order *Order) {
	var needCalculate bool
	if order.info != nil {
		needCalculate = dealer.CurrentHeight() > order.info.ExpiredOn
	}
	err := dealer.client.PlaceStorageOrder(order.cid, order.fileSize, 0, needCalculate)
	if err != nil {
		order.SetStatus(OrderStatusRetry)
		if order.ErrorFound(err) >= 3 {
			order.SetStatus(OrderStatusError)
		}
	} else {
		order.WaitingAt(dealer.CurrentHeight())
	}
}

func (dealer *Dealer) CurrentHeight() types.BlockNumber {
	dealer.mtx.Lock()
	defer dealer.mtx.Unlock()
	return dealer.currentHeight
}

func (dealer *Dealer) SetCurrentHeight(height types.BlockNumber) {
	dealer.mtx.Lock()
	defer dealer.mtx.Unlock()
	dealer.currentHeight = height
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
	dealer.SetCurrentHeight(height)
	dealer.Start()
	return &dealer, nil
}
