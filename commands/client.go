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
	"io"
	"os"
	"strings"
	"time"
)

type barReader struct {
	file io.Reader
	bar  *progressbar.ProgressBar
}

func (b barReader) Read(p []byte) (n int, err error) {
	_, _ = b.bar.Write(p)
	n, err = b.file.Read(p)
	if err != nil {
		_ = b.bar.Close()
	}
	return
}

type ItsClient interface {
	Cp(file cid.Cid, dir path.Path, info types.PinInfo) error
	Ls(dir path.Path) ([]types.LsFileInfo, error)
	Mv(from path.Path, to path.Path) error
	Rm(dir path.Path) error
	Mkdir(dir path.Path) error
	Pin(info types.PinInfo) error
	UnPin(file cid.Cid) error
	Stat(cid cid.Cid) (types.LsFileInfo, error)
}

type ClientCommand struct {
	client ItsClient
	ctx    context.Context
	ipfs   *httpapi.HttpApi
}

func (cmd *ClientCommand) Cp(filePath string, dir string, duplicate int, crust bool) error {
	var fileCid cid.Cid
	var err error
	fileCid, err = cmd.filePath2Cid(filePath)
	if err != nil {
		return err
	}
	return cmd.client.Cp(fileCid, path.Path(dir), types.PinInfo{
		Cid:      fileCid,
		PinCount: duplicate,
		Crust:    crust,
	})
}

func (cmd *ClientCommand) Ls(filePath string) ([]types.LsFileInfo, error) {
	var fileCid cid.Cid
	var err error
	if strings.HasPrefix(filePath, "/ipfs/") {
		filePath = strings.ReplaceAll(filePath, "/ipfs/", "")
		fileCid, err = cid.Decode(filePath)
		if err != nil {
			return nil, err
		}
		if info, err := cmd.client.Stat(fileCid); err != nil {
			return nil, err
		} else {
			return []types.LsFileInfo{info}, nil
		}
	} else {
		return cmd.client.Ls(path.Path(filePath))
	}
}

func (cmd *ClientCommand) Mv(from string, to string) error {
	return cmd.client.Mv(path.Path(from), path.Path(to))
}

func (cmd *ClientCommand) Rm(file string) error {
	return cmd.client.Rm(path.Path(file))
}

func (cmd *ClientCommand) Mkdir(file string) error {
	return cmd.client.Mkdir(path.Path(file))
}

func (cmd *ClientCommand) Pin(fileCid cid.Cid, duplicate int, crust bool) error {
	return cmd.client.Pin(types.PinInfo{
		Cid:      fileCid,
		PinCount: duplicate,
		Crust:    crust,
	})
}

func (cmd *ClientCommand) UnPin(fileCid cid.Cid) error {
	return cmd.client.UnPin(fileCid)
}

func (cmd *ClientCommand) filePath2Cid(filePath string) (fileCid cid.Cid, err error) {
	if strings.HasPrefix(filePath, "/ipfs/") {
		filePath = strings.ReplaceAll(filePath, "/ipfs/", "")
		fileCid, err = cid.Decode(filePath)
	} else {
		fileCid, err = cmd.ipfsUpload(filePath)
	}
	return
}

func (cmd *ClientCommand) ipfsUpload(path string) (cid.Cid, error) {
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

func NewClientCommand(ctx context.Context, client ItsClient, ipfs *httpapi.HttpApi) *ClientCommand {
	return &ClientCommand{
		client: client,
		ctx:    ctx,
		ipfs:   ipfs,
	}
}
