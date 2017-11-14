package archive

import (
	"bufio"
	"compress/gzip"
	"context"
	"io"
	"path"

	mdag "github.com/cc14514/go-ipfs/merkledag"
	tar "github.com/cc14514/go-ipfs/unixfs/archive/tar"
	uio "github.com/cc14514/go-ipfs/unixfs/io"

	node "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"
	dlog "debuglogger"
)

// DefaultBufSize is the buffer size for gets. for now, 1MB, which is ~4 blocks.
// TODO: does this need to be configurable?
//var DefaultBufSize = 1048576
//var DefaultBufSize = 262144 // 256k debug mem
var DefaultBufSize = 2*262144 // modify by liangc : block size 128k ,so 4 blocks = 512k

type identityWriteCloser struct {
	w io.Writer
}

func (i *identityWriteCloser) Write(p []byte) (int, error) {
	return i.w.Write(p)
}

func (i *identityWriteCloser) Close() error {
	return nil
}

// DagArchive is equivalent to `ipfs getdag $hash | maybe_tar | maybe_gzip`
func DagArchive(ctx context.Context, nd node.Node, name string, dag mdag.DAGService, archive bool, compression int) (io.Reader, error) {

	_, filename := path.Split(name)

	// need to connect a writer to a reader
	piper, pipew := io.Pipe()
	checkErrAndClosePipe := func(err error) bool {
		if err != nil {
			pipew.CloseWithError(err)
			return true
		}
		return false
	}

	// use a buffered writer to parallelize task
	bufw := bufio.NewWriterSize(pipew, DefaultBufSize)

	// compression determines whether to use gzip compression.
	maybeGzw, err := newMaybeGzWriter(bufw, compression)
	if checkErrAndClosePipe(err) {
		return nil, err
	}

	closeGzwAndPipe := func() {
		if err := maybeGzw.Close(); checkErrAndClosePipe(err) {
			return
		}
		if err := bufw.Flush(); checkErrAndClosePipe(err) {
			return
		}
		pipew.Close() // everything seems to be ok.
	}

	// XXXX archive == false , compression = 0 , 此处走的是 else 分支 >>>>
	dlog.Println("TODO 0 : >>>>>>>>>>>>>>>",archive,compression)
	if !archive && compression != gzip.NoCompression {
		// the case when the node is a file
		//dlog.Println("TODO 1 : >>>>>>>>>>>>>>>","the case when the node is a file")
		dagr, err := uio.NewDagReader(ctx, nd, dag)
		//dlog.Println("TODO 1 : <<<<<<<<<<<<<<<",err,dagr.Size())
		if checkErrAndClosePipe(err) {
			return nil, err
		}

		go func() {
			if _, err := dagr.WriteTo(maybeGzw); checkErrAndClosePipe(err) {
				return
			}
			closeGzwAndPipe() // everything seems to be ok
		}()
	} else {
		// the case for 1. archive, and 2. not archived and not compressed, in which tar is used anyway as a transport format
		dlog.Println("TODO 2 : >>>>>>>>>>>>>>>","construct the tar writer")
		// construct the tar writer
		w, err := tar.NewWriter(ctx, dag, archive, compression, maybeGzw)
		if checkErrAndClosePipe(err) {
			return nil, err
		}

		go func() {
			// write all the nodes recursively
			if err := w.WriteNode(nd, filename); checkErrAndClosePipe(err) {
				return
			}
			w.Close()         // close tar writer
			closeGzwAndPipe() // everything seems to be ok
			dlog.Println("TODO 2 : <<<<<<<<<<<<<<<")
		}()
	}

	return piper, nil
}

func newMaybeGzWriter(w io.Writer, compression int) (io.WriteCloser, error) {
	if compression != gzip.NoCompression {
		return gzip.NewWriterLevel(w, compression)
	}
	return &identityWriteCloser{w}, nil
}
