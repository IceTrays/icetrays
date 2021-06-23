package rpc

import (
	"bytes"
	"github.com/icetrays/icetrays/types"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-path"
	"io/ioutil"
	"net/http"
)

type ItsClient struct {
	url string
}

func (i ItsClient) Cp(file cid.Cid, dir path.Path, info types.PinInfo) error {
	panic("implement me")
}

func (i ItsClient) Ls(dir path.Path) ([]types.LsFileInfo, error) {
	panic("implement me")
}

func (i ItsClient) Mv(from path.Path, to path.Path) error {
	panic("implement me")
}

func (i ItsClient) Rm(dir path.Path) error {
	panic("implement me")
}

func (i ItsClient) Mkdir(dir path.Path) error {
	panic("implement me")
}

func (i ItsClient) Pin(info types.PinInfo) error {
	panic("implement me")
}

func (i ItsClient) UnPin(file cid.Cid) error {
	panic("implement me")
}

func (i ItsClient) Stat(cid cid.Cid) (types.LsFileInfo, error) {
	panic("implement me")
}

func (i ItsClient) request(body []byte) ([]byte, error) {
	response, err := http.Post(i.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	res, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return res, nil
}
