package crust

import "github.com/ipfs/go-cid"

type OrderClient interface {
	PlaceStorageOrder(fileCid cid.Cid, fileSize uint64, tip uint64) error
	GetFileInfo(fileCid cid.Cid) (*FileInfo, error)
}

type Dealer struct {
	client OrderClient
}
