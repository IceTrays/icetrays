package types

import (
	"github.com/ipfs/go-cid"
	"time"
)

type PinInfo struct {
	Cid      cid.Cid
	PinCount uint32
	Crust    bool
}

type LsFileInfo struct {
	Name      string
	Size      int64
	IsDir     bool
	PinNodes  []string
	CrustInfo InfoInCrust
}

type InfoInCrust struct {
	Expire  time.Time
	Replica int
}
