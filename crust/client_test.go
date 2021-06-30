package crust

import (
	"fmt"
	"github.com/ipfs/go-cid"
	"testing"
	"time"
	//ghash "github.com/centrifuge/go-substrate-rpc-client/v3/hash"
)

func TestClient_GetFileInfo(t *testing.T) {
	//NewBlake2b128NewBlake2b128Concat
	client, err := NewClient("wss://rocky-api.crust.network/", "tomorrow gun unfair damp crisp pet basket zone matrix kidney together april", time.Minute)
	if err != nil {
		panic(err)
	}
	ccid, _ := cid.Decode("bafzbeigai3eoy2ccc7ybwjfz5r3rdxqrinwi4rwytly24tdbh6yk7zslrm")
	info, err := client.GetFileInfo(ccid)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v", info)

	//"0x5ebf094108ead4fefa73f7a3b13cb4a7b3b78f30e9b952d60249b22fcdaaa76defa82b650f93c5afb8516d504a4759565a6b65444466783978675847486e7a4250374b6d336751526657665a7a414238325a3444536461"
}
