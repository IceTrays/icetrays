package types

import "github.com/ipfs/go-cid"

type RequestCpParams struct {
	Dir      string  `json:"dir"`
	File     cid.Cid `json:"file"`
	PinCount int     `json:"pin_count"`
	Crust    bool    `json:"crust"`
}

type RequestLsParams struct {
	Dir string `json:"dir"`
}

type RequestMvParams struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type RequestRmParams struct {
	Dir string `json:"dir"`
}

type RequestMkdirParams struct {
	Dir string `json:"dir"`
}

type RequestPinParams struct {
	File     cid.Cid `json:"file"`
	PinCount int     `json:"pin_count"`
	Crust    bool    `json:"crust"`
}

type RequestUnPinParams struct {
	File cid.Cid `json:"file"`
}
