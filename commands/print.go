package commands

import (
	"fmt"
	"github.com/icetrays/icetrays/types"
)

var sizeTail = []string{"B", "KB", "MB", "GB", "TB"}

func printLsFileInfo(info types.LsFileInfo) {
	var name string
	if info.IsDir {
		name = fmt.Sprintf("%c[1;;34m%s%c[0m", 0x1B, info.Name, 0x1B)
	} else {
		name = fmt.Sprintf("%c[1;;30m%s%c[0m", 0x1B, info.Name, 0x1B)
	}
	fmt.Printf("%s\t%s\tlocal-replica: %d\tcrust-replica: %d\n", name, SizeString(info.Size), len(info.PinNodes), info.CrustInfo.Replica)
}

func SizeString(size int64) string {
	if size < 0 {
		panic("size must not be negative")
	}
	var cur int
	var left = size
	var right int64
	for {
		if left < 1024 {
			break
		}
		left = left >> 10

		cur += 1
	}
	if cur != 0 {
		right = size>>(10*(cur-1)) - left<<10
		return fmt.Sprintf("%d.%03d %s", left, right*1000>>10, sizeTail[cur])
	} else {
		return fmt.Sprintf("%d %s", left, sizeTail[cur])
	}
}
