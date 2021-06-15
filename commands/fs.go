package commands

import (
	"context"
	"fmt"
	files "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/schollz/progressbar/v3"

	//"github.com/schollz/progressbar/v3"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"os"
)

type object struct {
	Hash string
}

//https://github.com/schollz/progressbar

var (
	app          = kingpin.New("itsc", "command-line of icetrays client.")
	iceTraysHome = app.Flag("home", "home path").Default("./").Envar("ICETRAYS_HOME").String()

	add             = app.Command("add", "Register a new user.")
	addDir          = add.Arg("dir", "dir to add").Required().String()
	addPath         = add.Arg("path", "file or dir path").Required().String()
	addPinDuplicate = add.Flag("pin", "pin duplicate").Default("-1").Int()
	addUseCrust     = add.Flag("crust", "use crust network").Default("false").Bool()
)

type BarReader struct {
	file io.Reader
	bar  io.Writer
}

func (b BarReader) Read(p []byte) (n int, err error) {
	_, _ = b.bar.Write(p)
	return b.file.Read(p)
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
	bar := progressbar.DefaultBytes(
		info.Size(),
		fmt.Sprintf("uploading %s", path),
	)
	fr := files.NewReaderFile(BarReader{f, bar})
	//d := files.NewMapDirectory(map[string]files.Node{"": fr}) // unwrapped on the other side

	//fileReader := files.NewMultiFileReader(d, false)
	//bs, err := ioutil.ReadAll(fileReader)
	//fmt.Println(string(bs))
	//return "", err

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
