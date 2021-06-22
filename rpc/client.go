package rpc

import (
	"context"
	"github.com/icetrays/icetrays/types"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-path"
)

type ItsClient struct {
}

func (i ItsClient) Cp(ctx context.Context, file cid.Cid, dir path.Path, info types.PinInfo) error {
	panic("implement me")
}

func (i ItsClient) Ls(ctx context.Context, dir path.Path) ([]types.LsFileInfo, error) {
	panic("implement me")
}

func (i ItsClient) Mv(ctx context.Context, from path.Path, to path.Path) error {
	panic("implement me")
}

func (i ItsClient) Rm(ctx context.Context, dir path.Path) error {
	panic("implement me")
}

func (i ItsClient) Mkdir(ctx context.Context, dir path.Path) error {
	panic("implement me")
}

func (i ItsClient) Pin(ctx context.Context, info types.PinInfo) error {
	panic("implement me")
}

func (i ItsClient) UnPin(ctx context.Context, file cid.Cid) error {
	panic("implement me")
}

func (i ItsClient) Stat(ctx context.Context, cid cid.Cid) (types.LsFileInfo, error) {
	panic("implement me")
}
