package helpservice

import (
	dlog "debuglogger"
	"github.com/cc14514/go-ipfs/core/coreunix"
	"context"
	"time"
)

func (p *HelpService) startHandler() {
	for msg := range p.msgCh {
		dlog.Println("helpservice_handler_msg :", msg)
		p.doHandler(msg)
	}
}

func (p *HelpService) doHandler(msg string) error {
	node := p.node
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mtype, fileid := getFilePath(msg)
	dlog.Println("👮 ========>", mtype, fileid)
	switch mtype {
	//TODO other msg type
	default:
		go func() {
			select {
			case <-time.After(15 * time.Second):
				dlog.Println("error : help_cat_timeout ,", fileid)
				cancel()
			}
		}()
		read, err := coreunix.Cat(ctx, node, fileid)
		if err != nil {
			dlog.Println("❌", err)
			return err
		}
		mem := make([]byte, 1024)
		t := 0
		for {
			i, e := read.Read(mem)
			if e == nil {
				t += i
			} else {
				break
			}
		}
		dlog.Println("help_cat_total_byte:", t, " ,", fileid)
	}
	return nil
}

func getFilePath(msg string) (mtype string, fileid string) {
	bmsg := []byte(msg)
	mtype = string(bmsg[0:2])
	fileid = string(bmsg[2:len(bmsg)])
	return
}
