package wal

import (
	"MyRaft/pkg/crc"
	raftPB "MyRaft/raft/raftpb"
	walPB "MyRaft/server/storage/wal/walpb"
	"bufio"
	"encoding/binary"
	"google.golang.org/protobuf/proto"
	"hash"
	"io"
	"sync"
)

const minSectorSize = 512

type decoder struct {
	mu  sync.Mutex
	brs []*bufio.Reader

	// lastValidOff file offset following the last valid decoded record
	lastValidOff int64
	crc          hash.Hash32
}

func newDecoder(r ...io.Reader) *decoder {
	readers := make([]*bufio.Reader, len(r))
	for i := range r {
		readers[i] = bufio.NewReader(r[i])
	}
	return &decoder{
		brs: readers,
		crc: crc.New(0, crcTable),
	}
}

func (d *decoder) decode(rec *walPB.Record) error {
	rec.Reset()
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.decodeRecord(rec)
}

const maxWALEntrySizeLimit = int64(10 * 1024 * 1024)

func (d *decoder) decodeRecord(rec *walPB.Record) error {
	if len(d.brs) == 0 {
		return io.EOF
	}
	l, err := readInt64(d.brs[0])
	if err == io.EOF || (err == nil && l == 0) {
		// hit end of file or preallocated space
		d.brs = d.brs[1:]
		if len(d.brs) == 0 {
			return io.EOF
		}
		d.lastValidOff = 0
		return d.decodeRecord(rec)
	}
	if err != nil {
		return err
	}

	recBytes, padBytes := decodeFrameSize(l)
	if recBytes >= maxWALEntrySizeLimit-padBytes{
		return ErrMaxWalEntrySizeLimitExceeded
	}
	data := make([]byte, recBytes+padBytes)
	if _, err = io.ReadFull(d.brs[0], data); err != nil{
		if err == io.EOF{
			err = io.ErrUnexpectedEOF
		}
		return err
	}

	if err := proto.Unmarshal(data[:recBytes], rec); err != nil{
		if d.
		return err
	}
	return nil

}
func (d *decoder) isTornEntry(data []byte) bool {
	if len(d.brs) != 1{
		return false
	}
	fileOff := d.lastValidOff + frameSizeBytes
}
func decodeFrameSize(lenField int64) (recBytes int64, padBytes int64) {
	recBytes = int64(uint64(lenField) &^ (uint64(0xff) << 56))
	if lenField < 0 {
		padBytes = int64((uint64(lenField) >> 56 & 0x7))
	}
	return recBytes, recBytes
}

func (d *decoder) lastOffset() int64 {
	return d.lastValidOff
}

func mustUnmarshalEntry(d []byte) raftPB.Entry {
	var e raftPB.Entry
	proto.Unmarshal(d, &e)
	return e
}

func mustUnmarshalState(d []byte) raftPB.HardState {
	var s raftPB.HardState
	proto.Unmarshal(d, &s)
	return s
}
func readInt64(r io.Reader) (int64, error) {
	var n int64
	err := binary.Read(r, binary.LittleEndian, &n)
	return n, err
}
