package commands

import (
	"context"
	"fmt"
	"github.com/icetrays/icetrays/types"
	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/go-path"
	"github.com/schollz/progressbar/v3"
	"os"
	"strings"
	"time"
)

type ItsClient interface {
	Cp(file cid.Cid, dir path.Path, info types.PinInfo) error
	Ls(dir path.Path) error
	Mv(from path.Path, to path.Path) error
	Rm(dir path.Path) error
	Mkdir(dir path.Path) error
	Pin(info types.PinInfo) error
}

type Command struct {
	client ItsClient
	ctx    context.Context
	ipfs   *httpapi.HttpApi
}

func (cmd *Command) Cp(filePath string, dir path.Path, duplicate int, crust bool) error {
	var fileCid cid.Cid
	var err error
	if !strings.HasPrefix(filePath, "/ipfs/") {
		fileCid, err = cmd.ipfsUpload(filePath)
	} else {
		fileCid, err = cid.Decode(filePath)
	}
	if err != nil {
		return err
	}
	return cmd.client.Cp(fileCid, dir, types.PinInfo{
		PinCount: duplicate,
		Crust:    crust,
	})
}

func (cmd *Command) ipfsUpload(path string) (cid.Cid, error) {
	f, err := os.Open(path)
	if err != nil {
		return cid.Undef, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return cid.Undef, err
	}
	bar := progressbar.NewOptions64(
		info.Size(),
		progressbar.OptionSetDescription(path),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			_, _ = fmt.Fprint(os.Stdout, "\n")
		}),
		progressbar.OptionSpinnerType(15),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerPadding: " ",
			BarStart:      "|",
			BarEnd:        "|",
			SaucerHead:    ">",
		}),
	)
	_ = bar.RenderBlank()

	fr := files.NewReaderFile(barReader{f, bar})

	re, err := cmd.ipfs.Unixfs().Add(cmd.ctx, fr)
	if err != nil {
		return cid.Undef, err
	}
	return re.Cid(), nil
}
