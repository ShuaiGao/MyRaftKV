package wal

import (
	"fmt"
	"testing"
)

func Test_decodeFrameSize(t *testing.T) {
	lenField := 13
	fmt.Printf("%x\n", uint64(^(uint64(0xff) << 56)))
	recBytes := int64(uint64(lenField) &^ (uint64(0xff) << 56))
	fmt.Println(recBytes)

	a, b := decodeFrameSize(12)
	fmt.Println(a)
	fmt.Println(b)
}
