package commands

import (
	"fmt"
	"github.com/icetrays/icetrays/types"
	"github.com/multiformats/go-multiaddr"
	"testing"
)

func TestSizeString(t *testing.T) {
	//34
	printLsFileInfo(types.LsFileInfo{
		Name:     "test1",
		Size:     1111024,
		IsDir:    true,
		PinNodes: []string{"", "", ""},
		CrustInfo: types.InfoInCrust{
			Replica: 10,
		},
	})
	fmt.Println(SizeString(1))
	fmt.Println(SizeString(1024))
	fmt.Println(SizeString(10024))
	fmt.Println(SizeString(11024))
	fmt.Println(SizeString(1111024))
	ma, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/5001")
	p := ma.Protocols()[0]
	fmt.Println(string(p.VCode))
	fmt.Println()
}
