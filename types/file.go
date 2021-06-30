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
	Name      string      `json:"name"`
	Size      int64       `json:"size"`
	IsDir     bool        `json:"is_dir"`
	PinNodes  []string    `json:"pin_nodes"`
	CrustInfo InfoInCrust `json:"crust_info"`
}

type InfoInCrust struct {
	Expire  time.Time
	Replica int
}
