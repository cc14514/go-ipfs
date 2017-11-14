package ping

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

	u "gx/ipfs/QmSU6eubNdhXjFBJBSksTp8kv8YRub8mGAPv8tVJHmL2EU/go-ipfs-util"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	"gx/ipfs/QmUwW8jMQDxXhLD2j4EfWqLEMX3MsvyWcWGvJPVDh1aTmu/go-libp2p-host"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	inet "gx/ipfs/QmahYsGWry85Y7WUe2SX5G4JkH2zifEQAUtJVLZ24aC9DF/go-libp2p-net"
	dlog "debuglogger"
)

var log = logging.Logger("ping")

const PingSize = 48

//const PingSize = 32

const ID = "/ipfs/ping/1.0.0"

const pingTimeout = time.Second * 60

type PingService struct {
	Host host.Host
	// liangc :增加一个管道，从外面传过来，求助的消息通过管道传递到相应的 handler
	helpMsgCh chan []byte
}

// add by liangc
func NewPingService2(h host.Host, hmCh chan []byte) *PingService {
	ps := &PingService{Host: h, helpMsgCh: hmCh}
	h.SetStreamHandler(ID, ps.PingHandler)
	return ps
}
func NewPingService(h host.Host) *PingService {
	ps := &PingService{Host: h}
	h.SetStreamHandler(ID, ps.PingHandler)
	return ps
}

func (p *PingService) PingHandler(s inet.Stream) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buf := make([]byte, PingSize)

	timer := time.NewTimer(pingTimeout)
	defer timer.Stop()

	go func() {
		select {
		case <-timer.C:
		case <-ctx.Done():
		}

		s.Close()
	}()

	for {
		_, err := io.ReadFull(s, buf)
		// add by liangc H:file_hash 这个是需要帮忙寻找资源的消息，其他类型待扩展
		dlog.Printf(">--ping--> %s", string(buf))
		switch string(buf[0:2]) {
		case "H:":
			dlog.Printf("helpMsgCh <- %s", string(buf))
			p.helpMsgCh <- buf
		}

		if err != nil {
			log.Debug(err)
			return
		}

		_, err = s.Write(buf)
		if err != nil {
			log.Debug(err)
			return
		}

		timer.Reset(pingTimeout)
	}
}

func (ps *PingService) Ping(ctx context.Context, p peer.ID) (<-chan time.Duration, error) {
	return ps.PingWithMsg(ctx, p, nil)
}

func (ps *PingService) PingWithMsg(ctx context.Context, p peer.ID, msg []byte) (<-chan time.Duration, error) {
	s, err := ps.Host.NewStream(ctx, p, ID)
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
				t, err := ping(s, msg)
				if err != nil {
					log.Debugf("ping error: %s", err)
					return
				}

				ps.Host.Peerstore().RecordLatency(p, t)
				select {
				case out <- t:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

func ping(s inet.Stream, msg []byte) (time.Duration, error) {
	buf := make([]byte, PingSize)
	if msg == nil {
		u.NewTimeSeededRand().Read(buf)
	} else {
		buf = msg[:]
	}
	before := time.Now()
	_, err := s.Write(buf)
	dlog.Printf("<--ping--< %d: %s , err=%v", len(buf), string(buf), err)
	if err != nil {
		return 0, err
	}

	rbuf := make([]byte, PingSize)
	_, err = io.ReadFull(s, rbuf)
	if err != nil {
		return 0, err
	}

	if !bytes.Equal(buf, rbuf) {
		return 0, errors.New("ping packet was incorrect!")
	}

	return time.Since(before), nil
}
