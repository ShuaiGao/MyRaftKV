package ioutil

import (
	"fmt"
	"io"
)

type ReaderAndCloser struct {
	io.Reader
	io.Closer
}

var (
	ErrShortRead = fmt.Errorf("ioutil: short read")
	ErrExpectEOF = fmt.Errorf("ioutil: expect EOF")
)

func NewExactReadCloser(rc io.ReadCloser, totalBytes int64) io.ReadCloser {
	return &exactReadCloser{
		rc:         rc,
		totalBytes: totalBytes,
	}
}

type exactReadCloser struct {
	rc         io.ReadCloser
	br         int64
	totalBytes int64
}

func (e *exactReadCloser) Read(p []byte) (n int, err error) {
	n, err = e.rc.Read(p)
	e.br += int64(n)
	if e.br > e.totalBytes {
		return 0, ErrExpectEOF
	}
	if e.br < e.totalBytes && n == 0 {
		return 0, ErrShortRead
	}
	return n, err
}

func (e *exactReadCloser) Close() error {
	if err := e.rc.Close(); err != nil {
		return nil
	}
	if e.br < e.totalBytes {
		return ErrShortRead
	}
	return nil
}
