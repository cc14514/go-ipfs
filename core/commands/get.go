package commands

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	gopath "path"
	"strings"

	"gx/ipfs/QmeWjRodbcZFKe5tMN7poEx3izym6osrLSnTLf9UjJZBbs/pb"

	cmds "github.com/cc14514/go-ipfs/commands"
	"github.com/cc14514/go-ipfs/core"
	dag "github.com/cc14514/go-ipfs/merkledag"
	"github.com/cc14514/go-ipfs/path"
	"github.com/cc14514/go-ipfs/thirdparty/tar"
	uarchive "github.com/cc14514/go-ipfs/unixfs/archive"
	"time"
	dlog "debuglogger"
	"github.com/cc14514/go-ipfs/helpservice"
)

var ErrInvalidCompressionLevel = errors.New("Compression level must be between 1 and 9")

var GetCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Download IPFS objects.",
		ShortDescription: `
Stores to disk the data contained an IPFS or IPNS object(s) at the given path.

By default, the output will be stored at './<ipfs-path>', but an alternate
path can be specified with '--output=<path>' or '-o=<path>'.

To output a TAR archive instead of unpacked files, use '--archive' or '-a'.

To compress the output with GZIP compression, use '--compress' or '-C'. You
may also specify the level of compression by specifying '-l=<1-9>'.
`,
	},

	Arguments: []cmds.Argument{
		cmds.StringArg("ipfs-path", true, false, "The path to the IPFS object(s) to be outputted.").EnableStdin(),
	},
	Options: []cmds.Option{
		cmds.StringOption("output", "o", "The path where the output should be stored."),
		cmds.BoolOption("archive", "a", "Output a TAR archive.").Default(false),
		cmds.BoolOption("compress", "C", "Compress the output with GZIP compression.").Default(false),
		cmds.IntOption("compression-level", "l", "The level of compression (1-9).").Default(-1),
	},
	PreRun: func(req cmds.Request) error {
		_, err := getCompressOptions(req)
		return err
	},
	Run: func(req cmds.Request, res cmds.Response) {
		if len(req.Arguments()) == 0 {
			res.SetError(errors.New("not enough arugments provided"), cmds.ErrClient)
			return
		}

		cmplvl, err := getCompressOptions(req)
		if err != nil {
			res.SetError(err, cmds.ErrClient)
			return
		}

		node, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}
		p := path.Path(req.Arguments()[0])
		ctx := req.Context()

		// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
		// add by liangc
		// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
		// 指定时间内，没有解除阻塞，这个地方要去请其他节点帮忙了，发一个广播
		cancelCh := make(chan int, 1)
		go func() {
			select {
			case <-cancelCh:
				//撤销
				dlog.Println("****** cancel_1 ******")
			case <-time.After(120 * time.Second):
				//触发
				dlog.Println("<<<<<< help_me_event >>>>>> ")
				hs := helpservice.GetInstance(node)
				//寻找帮忙的节点
				//TODO 简单点，就找 peers 们帮忙吧,但是要尽量排除 种子节点
				//TODO 相同网段的 peers 都应该排除在外，否则会造成不必要的阻塞
				peerids := hs.MyPeerIDs()
				helpMsg := "00" + p.String()
				for i, pid := range peerids {
					o, e := hs.Send(ctx, pid, helpMsg)
					if e != nil {
						dlog.Println("send_help_msg_err", i, pid, e)
					} else {
						select {
						case t := <-o:
							dlog.Println("send_help_msg_ok", i, pid, t)
						case <-time.After(5 * time.Second):
							dlog.Println("send_help_msg_timeout", i, pid)
						case <-cancelCh:
							//撤销
							dlog.Println("****** cancel_2 ******")
						}
					}
				}
			}
		}()
		dlog.Printf("--> %s", "resolve_wait")
		/*
		XXXX 通过下面的调试，可以看出 dn 中已经包含了第一级的全部分块信息，dn.links 是第一级分片列表,可以用来还原整个资源
			 而且在 Add 时，指明了 raw 的资源，在这里获取时也是 *dag.ProtoNode 类型,没有走 *dag.RawNode 分支。
			 结论是只要拿到 dn 对象，就可以得到整个资源，dn 对象是树根节点
		*/

		dn, err := core.Resolve(ctx, node.Namesys, node.Resolver, p)
		dlog.Printf("--> %s , err = %s", "resolve_over", err)
		cancelCh <- 0
		// <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
		// add by liangc
		// <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		//dlog.Println("TTTTTTTTTT>",dn.Links())
		//dlog.Println("TTTTTTTTTTTTTTTTTTTTT>",len(dn.Links()),dn.Cid(),dn.String())
		switch dn := dn.(type) {
		case *dag.ProtoNode:
			size, err := dn.Size()
			dlog.Println("TTTTTTTTTTTTTTTTTTTTTTTTTTT> 1 ProtoNode ",size,err)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			res.SetLength(size)
		case *dag.RawNode:
			dlog.Println("TTTTTTTTTTTTTTTTTTTTTTTTTTT> 2 RawNode ",len(dn.RawData()),err)
			res.SetLength(uint64(len(dn.RawData())))
		default:
			res.SetError(fmt.Errorf("'ipfs get' only supports unixfs nodes"), cmds.ErrNormal)
			return
		}

		archive, _, _ := req.Option("archive").Bool()
		/*
		TODO 这里的 reader 被封装成 标准 io.Reader ，但是是从分布式网络中取资源，需要深入探索内存使用量太高的问题

		TODO 此处的第二个问题是，Resolve 完成以后，依然会有读不到数据的情况，网络通为何数据不通？
		*/
		reader, err := uarchive.DagArchive(ctx, dn, p.String(), node.DAG, archive, cmplvl)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}
		res.SetOutput(reader)
	},
	PostRun: func(req cmds.Request, res cmds.Response) {
		if res.Output() == nil {
			return
		}
		outReader := res.Output().(io.Reader)
		res.SetOutput(nil)

		outPath, _, _ := req.Option("output").String()
		if len(outPath) == 0 {
			_, outPath = gopath.Split(req.Arguments()[0])
			outPath = gopath.Clean(outPath)
		}

		cmplvl, err := getCompressOptions(req)
		if err != nil {
			res.SetError(err, cmds.ErrClient)
			return
		}

		archive, _, _ := req.Option("archive").Bool()

		gw := getWriter{
			Out:         os.Stdout,
			Err:         os.Stderr,
			Archive:     archive,
			Compression: cmplvl,
			Size:        int64(res.Length()),
		}

		if err := gw.Write(outReader, outPath); err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}
	},
}

type clearlineReader struct {
	io.Reader
	out io.Writer
}

func (r *clearlineReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if err == io.EOF {
		// callback
		fmt.Fprintf(r.out, "\033[2K\r") // clear progress bar line on EOF
	}
	return
}

func progressBarForReader(out io.Writer, r io.Reader, l int64) (*pb.ProgressBar, io.Reader) {
	bar := makeProgressBar(out, l)
	barR := bar.NewProxyReader(r)
	return bar, &clearlineReader{barR, out}
}

func makeProgressBar(out io.Writer, l int64) *pb.ProgressBar {
	// setup bar reader
	// TODO: get total length of files
	bar := pb.New64(l).SetUnits(pb.U_BYTES)
	bar.Output = out

	// the progress bar lib doesn't give us a way to get the width of the output,
	// so as a hack we just use a callback to measure the output, then git rid of it
	bar.Callback = func(line string) {
		terminalWidth := len(line)
		bar.Callback = nil
		log.Infof("terminal width: %v\n", terminalWidth)
	}
	return bar
}

type getWriter struct {
	Out io.Writer // for output to user
	Err io.Writer // for progress bar output

	Archive     bool
	Compression int
	Size        int64
}

func (gw *getWriter) Write(r io.Reader, fpath string) error {
	if gw.Archive || gw.Compression != gzip.NoCompression {
		return gw.writeArchive(r, fpath)
	}
	return gw.writeExtracted(r, fpath)
}

func (gw *getWriter) writeArchive(r io.Reader, fpath string) error {
	// adjust file name if tar
	if gw.Archive {
		if !strings.HasSuffix(fpath, ".tar") && !strings.HasSuffix(fpath, ".tar.gz") {
			fpath += ".tar"
		}
	}

	// adjust file name if gz
	if gw.Compression != gzip.NoCompression {
		if !strings.HasSuffix(fpath, ".gz") {
			fpath += ".gz"
		}
	}

	// create file
	file, err := os.Create(fpath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(gw.Out, "Saving archive to %s\n", fpath)
	bar, barR := progressBarForReader(gw.Err, r, gw.Size)
	bar.Start()
	defer bar.Finish()

	_, err = io.Copy(file, barR)
	return err
}

func (gw *getWriter) writeExtracted(r io.Reader, fpath string) error {
	fmt.Fprintf(gw.Out, "Saving file(s) to %s\n", fpath)
	bar := makeProgressBar(gw.Err, gw.Size)
	bar.Start()
	defer bar.Finish()
	defer bar.Set64(gw.Size)

	extractor := &tar.Extractor{fpath, bar.Add64}
	return extractor.Extract(r)
}

func getCompressOptions(req cmds.Request) (int, error) {
	cmprs, _, _ := req.Option("compress").Bool()
	cmplvl, cmplvlFound, _ := req.Option("compression-level").Int()
	switch {
	case !cmprs:
		return gzip.NoCompression, nil
	case cmprs && !cmplvlFound:
		return gzip.DefaultCompression, nil
	case cmprs && cmplvlFound && (cmplvl < 1 || cmplvl > 9):
		return gzip.NoCompression, ErrInvalidCompressionLevel
	}
	return cmplvl, nil
}
