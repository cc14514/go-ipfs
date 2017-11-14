package io

import (
	"context"
	"errors"
	"fmt"
	"io"

	mdag "github.com/cc14514/go-ipfs/merkledag"
	ft "github.com/cc14514/go-ipfs/unixfs"
	ftpb "github.com/cc14514/go-ipfs/unixfs/pb"

	node "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	dlog "debuglogger"
)

var ErrIsDir = errors.New("this dag node is a directory")

var ErrCantReadSymlinks = errors.New("cannot currently read symlinks")

type DagReader interface {
	ReadSeekCloser
	Size() uint64
	CtxReadFull(context.Context, []byte) (int, error)
	Offset() int64
}

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
	io.WriterTo
}

// NewDagReader creates a new reader object that reads the data represented by
// the given node, using the passed in DAGService for data retreival
func NewDagReader(ctx context.Context, n node.Node, serv mdag.DAGService) (DagReader, error) {
	switch n := n.(type) {
	case *mdag.RawNode:
		//dlog.Println("<unixfs/io/dagreader.go>TTT[RawNode]TTTT> data_size =",len(n.RawData()),n.String())
		return NewBufDagReader(n.RawData()), nil
	case *mdag.ProtoNode:
		//dlog.Println("TTTT> data_size =",len(n.Data()),n.String())
		pb := new(ftpb.Data)
		if err := proto.Unmarshal(n.Data(), pb); err != nil {
			return nil, err
		}
		dlog.Println("TTTTTTTT> pb_data_size =",len(pb.GetData()),",pb_filesize =",pb.GetFilesize(),", type =",pb.GetType())
		// debug >>>>
		for i,s := range pb.GetBlocksizes() {
			dlog.Println(i,s)
		}
		// debug <<<<

		switch pb.GetType() {
		case ftpb.Data_Directory, ftpb.Data_HAMTShard:
			// Dont allow reading directories
			return nil, ErrIsDir
		case ftpb.Data_File, ftpb.Data_Raw:
			dlog.Println("TTTTTTTTTTTT> NewPBFileReader",pb.GetType())
			return NewPBFileReader(ctx, n, pb, serv), nil
		case ftpb.Data_Metadata:
			if len(n.Links()) == 0 {
				return nil, errors.New("incorrectly formatted metadata object")
			}
			child, err := n.Links()[0].GetNode(ctx, serv)
			if err != nil {
				return nil, err
			}

			childpb, ok := child.(*mdag.ProtoNode)
			if !ok {
				return nil, mdag.ErrNotProtobuf
			}
			return NewDagReader(ctx, childpb, serv)
		case ftpb.Data_Symlink:
			return nil, ErrCantReadSymlinks
		default:
			return nil, ft.ErrUnrecognizedType
		}
	default:
		return nil, fmt.Errorf("unrecognized node type")
	}
}
