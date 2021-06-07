package crust

import (
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/ipfs/go-cid"
)

type OrderStatus int

const (
	OrderStatusStart OrderStatus = iota
	OrderStatusWaiting
	OrderStatusAccepted
	OrderStatusRenew
	OrderStatusRetry
	OrderStatusError
)

type Order struct {
	cid        cid.Cid
	fileSize   uint64
	height     types.BlockNumber
	info       FileInfo
	status     OrderStatus
	retryTimes int
	err        error
}

func (order *Order) SetStatus(status OrderStatus) {
	order.status = status
}
