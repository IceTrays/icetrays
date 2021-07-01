package pinservice

import (
	"fmt"
	"sort"
	"testing"
)

func TestNodeList_Swap(t *testing.T) {
	n := NodeSelector{
		nodes: []string{"12D3KooWJGCfGHBWkFS3DgZQZ9WdAQ29BkuXXRcJhpYiWuqAPVVC", "12D3KooWH6rowPV3eVc3QjiMRDAWva5LpYrt2optPQZiwUm5jQ6c", "12D3KooWJbkbLcZas8LxmmLomJpKZ69oJA1CX8AeNcqPBphWVHgM"},
		cid:   "12D3KooWJGCfGHBWkFS3DgZQZ9WdAQ29BkuXXRcJhpYiWuqAPVVC",
	}
	sort.Sort(&n)
	fmt.Println(n.nodes)
}
