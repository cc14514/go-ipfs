package helpservice

import (
	"time"
	"bytes"
	"context"
	"io"
	"errors"
	dlog "debuglogger"
	//host "gx/ipfs/Qmc1XhrFEiSeBNn3mpfg6gEuYCt5im2gYmNVmncsvmpeAk/go-libp2p-host"
	//u "gx/ipfs/QmSU6eubNdhXjFBJBSksTp8kv8YRub8mGAPv8tVJHmL2EU/go-ipfs-util"
	inet "gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"github.com/cc14514/go-ipfs/core"
)

//packet format : TYPE[2]FILEID[46]
const PacketSize = 48

const ID = "/ipfs/help/1.0.0"

const msgTimeout = time.Second * 60

type HelpService struct {
	node  *core.IpfsNode
	msgCh chan string
}

var helpService *HelpService

func GetInstance(n *core.IpfsNode) *HelpService {
	if helpService != nil {
		return helpService
	}
	return NewHelpService(n)
}

func NewHelpService(n *core.IpfsNode) *HelpService {
	helpService = &HelpService{node: n, msgCh: make(chan string, 16)}
	n.PeerHost.SetStreamHandler(ID, helpService.MsgHandler)
	go helpService.startHandler()
	dlog.Println("--> liangc:help_service_started <--")
	return helpService
}

func (p *HelpService) MsgHandler(s inet.Stream) {
	buf := make([]byte, PacketSize)
	errCh := make(chan error, 1)
	defer close(errCh)
	timer := time.NewTimer(msgTimeout)
	defer timer.Stop()

	go func() {
		select {
		case <-timer.C:
			dlog.Println("help timeout")
			s.Reset()
		case err, ok := <-errCh:
			if ok {
				dlog.Println(err)
				if err == io.EOF {
					s.Close()
				} else {
					s.Reset()
				}
			} else {
				dlog.Println("help loop failed without error")
			}
		}
	}()

	for {
		_, err := io.ReadFull(s, buf)
		if err != nil {
			errCh <- err
			return
		}
		p.msgCh <- string(buf)
		if err != nil {
			//TODO execute handler , if the queue was full , return error
		}
		_, err = s.Write(buf)
		if err != nil {
			errCh <- err
			return
		}
		timer.Reset(msgTimeout)
	}
}

func (ps *HelpService) MyPeerIDs() []peer.ID {
	conns := ps.node.PeerHost.Network().Conns()
	if conns == nil {
		return nil
	}
	l := make([]peer.ID,len(conns))
	for i, c := range conns {
		pid := c.RemotePeer()
		l[i] = pid
	}
	return l
}

func (ps *HelpService) Send(ctx context.Context, p peer.ID, m string) (<-chan time.Duration, error) {
	s, err := ps.node.PeerHost.NewStream(ctx, p, ID)
	if err != nil {
		return nil, err
	}
	out := make(chan time.Duration)
	go func() {
		defer close(out)
		defer s.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				t, err := send(s, m)
				if err != nil {
					s.Reset()
					dlog.Printf("--> help error: %s", err)
					return
				}
				//ps.Host.Peerstore().RecordLatency(p, t)
				select {
				case out <- t:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	dlog.Println("🙏 ====>", m)
	return out, nil
}

func send(s inet.Stream, m string) (time.Duration, error) {
	buf := []byte(m)
	before := time.Now()
	_, err := s.Write(buf)
	if err != nil {
		return 0, err
	}

	rbuf := make([]byte, PacketSize)
	_, err = io.ReadFull(s, rbuf)
	if err != nil {
		return 0, err
	}

	if !bytes.Equal(buf, rbuf) {
		//TODO exception reason switch
		return 0, errors.New("help packet was incorrect!")
	}
	return time.Since(before), nil
}
