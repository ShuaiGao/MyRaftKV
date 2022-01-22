package wal

import (
	raftPB "MyRaft/raft/raftpb"
	"errors"
	"go.uber.org/zap"
	"hash/crc32"
	"os"
)

var (
	crcTable                        = crc32.MakeTable(crc32.Castagnoli)
	ErrMaxWalEntrySizeLimitExceeded = errors.New("wal: max entry size limit exceeded")
)

type WAL struct {
	lg  *zap.Logger
	dir string //

	// dirFile is fd for whe wal director for syncing on Rename
	dirFile *os.File

	metadata []byte           // metadata recorded at the head of each WAL
	state    raftPB.HardState // hardstate recorded at the head of WAL
}
