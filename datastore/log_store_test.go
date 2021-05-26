package datastore

import (
	"context"
	"fmt"
	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/multiformats/go-multiaddr"
	"testing"
)

func TestNewStableDB(t *testing.T) {
	addr, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/5001")
	api, _ := httpapi.NewApi(addr)
	id, _ := cid.Decode("QmTaGVtCuCCJdGErCh4haRUrZVr3kGLqAzkSunw9uBESJ3")
	node, _ := api.Dag().Get(context.Background(), id)
	fmt.Println(node.Stat())
}
