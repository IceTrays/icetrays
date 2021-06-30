package crust

import (
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/ipfs/go-cid"
	"sync"
)

type OrderStatus int

const (
	OrderStatusStart OrderStatus = iota
	OrderStatusWaiting
	OrderStatusAccepted
	OrderStatusRenew
	OrderStatusRetry
	OrderStatusError
	OrderStatusDiscard
)

type Order struct {
	cid        cid.Cid
	fileSize   uint64
	height     types.BlockNumber
	info       *FileInfo
	status     OrderStatus
	retryTimes int
	err        error
	mtx        sync.Mutex
}

func (order *Order) SetStatus(status OrderStatus) {
	order.mtx.Lock()
	defer order.mtx.Unlock()
	if status == OrderStatusAccepted {
		order.retryTimes = 0
	}
	order.status = status
}

func (order *Order) Status() OrderStatus {
	order.mtx.Lock()
	defer order.mtx.Unlock()
	return order.status
}

func (order *Order) WaitingAt(height types.BlockNumber) {
	order.mtx.Lock()
	defer order.mtx.Unlock()
	order.status = OrderStatusWaiting
	order.height = height
}

func (order *Order) ErrorFound(err error) (retryTimes int) {
	order.mtx.Lock()
	defer order.mtx.Unlock()
	order.err = err
	order.retryTimes += 1
	return order.retryTimes
}

func (order *Order) Error() error {
	order.mtx.Lock()
	defer order.mtx.Unlock()
	return order.err
}

func (order *Order) Cid() cid.Cid {
	order.mtx.Lock()
	defer order.mtx.Unlock()
	return order.cid
}

func (order *Order) Height() types.BlockNumber {
	order.mtx.Lock()
	defer order.mtx.Unlock()
	return order.height
}

func (order *Order) FileSize() uint64 {
	order.mtx.Lock()
	defer order.mtx.Unlock()
	return order.fileSize
}

func (order *Order) SetFileInfo(info *FileInfo) {
	order.mtx.Lock()
	defer order.mtx.Unlock()
	order.info = info
}

func (order *Order) FileInfo() FileInfo {
	order.mtx.Lock()
	defer order.mtx.Unlock()
	return *order.info
}
