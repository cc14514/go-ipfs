// package chunk implements streaming block splitters
package chunk

import (
	"io"

	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	"sync/atomic"
	"fmt"
)

var log = logging.Logger("chunk")

var DefaultBlockSize int64 = 1024 * 128
//var DefaultBlockSize int64 = 1024 * 256

type Splitter interface {
	Reader() io.Reader
	NextBytes() ([]byte, error)
}

type SplitterGen func(r io.Reader) Splitter

func DefaultSplitter(r io.Reader) Splitter {
	return NewSizeSplitter(r, DefaultBlockSize)
}

func SizeSplitterGen(size int64) SplitterGen {
	return func(r io.Reader) Splitter {
		return NewSizeSplitter(r, size)
	}
}

func Chan(s Splitter) (<-chan []byte, <-chan error) {
	out := make(chan []byte)
	errs := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errs)

		// all-chunks loop (keep creating chunks)
		for {
			b, err := s.NextBytes()
			if err != nil {
				errs <- err
				return
			}

			out <- b
		}
	}()
	return out, errs
}

type sizeSplitterv2 struct {
	r    io.Reader
	size int64
	err  error
}

func NewSizeSplitter(r io.Reader, size int64) Splitter {
	return &sizeSplitterv2{
		r:    r,
		size: size,
	}
}
var total = atomic.LoadInt64(new(int64))
func (ss *sizeSplitterv2) NextBytes() ([]byte, error) {
	if ss.err != nil {
		return nil, ss.err
	}
	total = atomic.AddInt64(&total,ss.size)
	m := total/1024/1024
	fmt.Printf("\r\b>>> (ss *sizeSplitterv2) NextBytes() : size = %d, total = %d MB",ss.size,m)
	buf := make([]byte, ss.size)
	n, err := io.ReadFull(ss.r, buf)
	if err == io.ErrUnexpectedEOF {
		ss.err = io.EOF
		err = nil
	}
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

func (ss *sizeSplitterv2) Reader() io.Reader {
	return ss.r
}
