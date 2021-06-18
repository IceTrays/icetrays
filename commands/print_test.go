package commands

import (
	"fmt"
	"github.com/icetrays/icetrays/types"
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
}
