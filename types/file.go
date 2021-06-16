package types

import "github.com/ipfs/go-cid"

type PinInfo struct {
	file     cid.Cid
	PinCount int
	Crust    bool
}
