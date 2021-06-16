package commands

import (
	"context"
	"fmt"
	files "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/schollz/progressbar/v3"
	"time"

	//"github.com/schollz/progressbar/v3"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"os"
)

var (
	app          = kingpin.New("itsc", "command-line of icetrays client.")
	iceTraysHome = app.Flag("home", "home path").Default("./").Envar("ICETRAYS_HOME").String()

	add             = app.Command("add", "Register a new user.")
	addDir          = add.Arg("dir", "dir to add").Required().String()
	addPath         = add.Arg("path", "file or dir path").Required().String()
	addPinDuplicate = add.Flag("pin", "pin duplicate").Default("-1").Int()
	addUseCrust     = add.Flag("crust", "use crust network").Default("false").Bool()
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

func FsAdd(s *httpapi.HttpApi, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
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

	re, err := s.Unixfs().Add(context.Background(), fr)
	if err != nil {
		panic(err)
	}
	fmt.Println(re.Cid().String())
	return "", err
}

func FsTest() {
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/5001")
	if err != nil {
		panic(err)
	}
	api, err := httpapi.NewApi(addr)
	if err != nil {
		panic(err)
	}

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	// Register user
	case add.FullCommand():
		fmt.Println(FsAdd(api, *addPath))

	}
}
