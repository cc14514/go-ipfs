package helpservicechannel

import (
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"context"
	"time"
)

type MyPeers struct {
	rtnCh chan []peer.ID
}

func NewMyPeers(rtnCh chan []peer.ID) *MyPeers {
	return &MyPeers{rtnCh}
}
func (self *MyPeers) GetRtnCh() chan []peer.ID {
	return self.rtnCh
}

type HelpMsgRtn struct {
	TtlCh <- chan time.Duration
	Error error
}

type HelpMsg struct {
	ctx   context.Context
	p     peer.ID
	m     string
	rtnCh chan HelpMsgRtn
}

func NewHelpMsg(ctx context.Context, p peer.ID, m string, rtnCh chan HelpMsgRtn) *HelpMsg {
	return &HelpMsg{
		ctx:   ctx,
		p:     p,
		m:     m,
		rtnCh: rtnCh,
	}
}

func (self *HelpMsg) GetCtx() context.Context {
	return self.ctx
}
func (self *HelpMsg) GetPeerID() peer.ID {
	return self.p
}
func (self *HelpMsg) GetMessage() string {
	return self.m
}
func (self *HelpMsg) GetRtnCh() chan HelpMsgRtn {
	return self.rtnCh
}
func (self *HelpMsg) String() string {
	return self.p.String() + " :: " + self.m
}

var MyPeersCh = make(chan *MyPeers)
var HelpCh = make(chan *HelpMsg)
