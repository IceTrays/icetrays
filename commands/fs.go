package commands

import (
	"context"
	"errors"
	"fmt"
	"github.com/icetrays/icetrays/types"
	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/go-path"
	ma "github.com/multiformats/go-multiaddr"
	//"github.com/schollz/progressbar/v3"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
)

//var (
//	iceTraysHome = "home"
//	add = "add"
//	addDir = "dir"
//	addPath = "path"
//	addPinDuplicate = "pin"
//	addUseCrust = "crust"
//)

var (
	app          = kingpin.New("itsc", "command-line of icetrays client.")
	iceTraysHome = app.Flag("home", "home path").Default("./").Envar("ICETRAYS_HOME").String()

	add             = app.Command("add", "add new file to icetrays")
	addPath         = add.Arg("file", "file or dir path").Required().String()
	addDir          = add.Arg("dir", "dir to add").Required().String()
	addPinDuplicate = add.Flag("pin", "pin duplicate").Default("-1").Int()
	addUseCrust     = add.Flag("crust", "use crust network").Default("false").Bool()
)

type mockClient struct {
}

func (m mockClient) Cp(file cid.Cid, dir path.Path, info types.PinInfo) error {
	if dir == "123" {
		return errors.New("test error")
	}
	return nil
}

func (m mockClient) Ls(dir path.Path) ([]types.LsFileInfo, error) {
	panic("implement me")
}

func (m mockClient) Mv(from path.Path, to path.Path) error {
	panic("implement me")
}

func (m mockClient) Rm(dir path.Path) error {
	panic("implement me")
}

func (m mockClient) Mkdir(dir path.Path) error {
	panic("implement me")
}

func (m mockClient) Pin(info types.PinInfo) error {
	panic("implement me")
}

func (m mockClient) UnPin(file cid.Cid) error {
	panic("implement me")
}

func (m mockClient) Stat(cid cid.Cid) (types.LsFileInfo, error) {
	panic("implement me")
}

func Run() {
	intrh, ctx := SetupInterruptHandler(context.Background())
	defer intrh.Close()

	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/5001")
	if err != nil {
		panic(err)
	}
	api, err := httpapi.NewApi(addr)
	if err != nil {
		panic(err)
	}
	cmd := NewClientCommand(ctx, &mockClient{}, api)

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	case add.FullCommand():
		err = cmd.Cp(*addPath, *addDir, *addPinDuplicate, *addUseCrust)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
			os.Exit(1)
		}

	}
}
