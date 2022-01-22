package wal

import (
	"MyRaft/client/pkg/fileutil"
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

type filePipeline struct {
	lg    *zap.Logger
	dir   string
	size  int64
	count int

	filec chan *fileutil.LockedFile
	errc  chan error
	donec chan struct{}
}

func newFilePipeline(lg *zap.Logger, dir string, fileSize int64) *filePipeline {
	if lg == nil {
		lg = zap.NewNop()
	}
	fp := &filePipeline{
		lg:    lg,
		dir:   dir,
		size:  fileSize,
		filec: make(chan *fileutil.LockedFile),
		errc:  make(chan error, 1),
		donec: make(chan struct{}),
	}
	go fp.run()
	return fp
}

func (fp *filePipeline) alloc() (f *fileutil.LockedFile, err error) {
	fpath := filepath.Join(fp.dir, fmt.Sprintf("%d.tmp", fp.count%2))
	if f, err = fileutil.LockFile()
}
func (fp *filePipeline) run() {
	defer close(fp.errc)
	for{
		f, err := fp.alloc()
		if err != nil{
			return
		}
		fp.errc <- err
		select {
		case fp.filec <-f:
			case <-fp.donec:
				os.Remove(f.Name())
				f.Close()
				return
		}
	}
}
