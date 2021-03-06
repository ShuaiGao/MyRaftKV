package crc

import (
	"hash"
	"hash/crc32"
)

const Size = 4

type digest struct {
	crc uint32
	tab *crc32.Table
}

func (d digest) Write(p []byte) (n int, err error) {
	d.crc = crc32.Update(d.crc, d.tab, p)
	return len(p), nil
}

func (d *digest) Sum(in []byte) []byte {
	s := d.Sum32()
	return append(in, byte(s>>24), byte(s>>16), byte(s>>8), byte(s))
}

func (d digest) Reset() {
	d.crc = 0
}

func (d *digest) Size() int {
	return Size
}

func (d *digest) BlockSize() int {
	return 1
}

func (d *digest) Sum32() uint32 {
	return d.crc
}

func New(prev uint32, tab *crc32.Table) hash.Hash32 { return &digest{prev, tab} }
