package crust

import "github.com/ipfs/go-cid"


type OrderStatus int

const (
	OrderStatusReady OrderStatus = iota
	OrderStatusAccepted
	OrderStatusRenew
	OrderStatusRetry
	OrderStatusError
)

type Order struct {
	cid cid.Cid
	info FileInfo
	status OrderStatus
}
