package state

import (
	"fmt"
	ds "github.com/ipfs/go-datastore"
	"testing"
)

func TestKey(t *testing.T) {
	sk := ds.KeyWithNamespaces([]string{PinUnfinished, "xxx"})
	n := sk.Namespaces()
	for _, k := range n {
		fmt.Println(k)

	}
}
