package consensus

import (
	"bytes"
	"fmt"
)

func Min(r1, r2 int) int {
	if r1 < r2 {
		return r1
	}
	return r2

}

func BytesToBinaryString(s string) string {
	bs := []byte(s)
	buf := bytes.NewBuffer([]byte{})
	for _, v := range bs {
		buf.WriteString(fmt.Sprintf("%08b", v))
	}
	return buf.String()
}
