package kfake

import (
	"encoding/binary"
	"io"
	"net"
	"time"

	"github.com/twmb/franz-go/pkg/kbin"
	"github.com/twmb/franz-go/pkg/kmsg"
)

type (
	clientConn struct {
		c      *Cluster
		b      *broker
		conn   net.Conn
		respCh chan clientResp
		done   chan struct{} // closed when read() returns

		saslStage saslStage
		s0        *scramServer0
		user      string // authenticated user, set after SASL completes
	}

	clientReq struct {
		cc        *clientConn
		kreq      kmsg.Request
		at        time.Time
		cid       string
		corr      int32
		seq       uint32
		topicMeta topicMetaSnap // snapshot for KIP-848 consumer group assignment
	}

	clientResp struct {
		kresp kmsg.Response
		corr  int32
		err   error
		seq   uint32
	}
)

func (creq *clientReq) empty() bool { return creq == nil || creq.cc == nil || creq.kreq == nil }

func (cc *clientConn) read() {
	defer close(cc.done)
	defer cc.conn.Close()

	type read struct {
		body []byte
		err  error
	}
	var (
		who    = cc.conn.RemoteAddr()
		size   = make([]byte, 4)
		readCh = make(chan read, 1)
		seq    uint32
	)
	for {
		go func() {
			if _, err := io.ReadFull(cc.conn, size); err != nil {
				readCh <- read{err: err}
				return
			}
			body := make([]byte, binary.BigEndian.Uint32(size))
			_, err := io.ReadFull(cc.conn, body)
			readCh <- read{body: body, err: err}
		}()

		var read read
		select {
		case <-cc.c.die:
			return
		case read = <-readCh:
		}

		if err := read.err; err != nil {
			return
		}

		var (
			body     = read.body
			reader   = kbin.Reader{Src: body}
			key      = reader.Int16()
			version  = reader.Int16()
			corr     = reader.Int32()
			clientID = reader.NullableString()
			kreq     = kmsg.RequestForKey(key)
		)
		kreq.SetVersion(version)
		if kreq.IsFlexible() {
			kmsg.SkipTags(&reader)
		}
		if err := kreq.ReadFrom(reader.Src); err != nil {
			cc.c.cfg.logger.Logf(LogLevelDebug, "client %s unable to parse request (key=%d, version=%d): %v", who, key, version, err)
			return
		}

		// Within Kafka, a null client ID is treated as an empty string.
		var cid string
		if clientID != nil {
			cid = *clientID
		}

		select {
		case cc.c.reqCh <- &clientReq{cc: cc, kreq: kreq, at: time.Now(), cid: cid, corr: corr, seq: seq}:
			seq++
		case <-cc.c.die:
			return
		}
	}
}

func (cc *clientConn) write() {
	defer cc.conn.Close()

	var (
		who     = cc.conn.RemoteAddr()
		writeCh = make(chan error, 1)
		buf     []byte
		seq     uint32

		// If a request is by necessity slow (join&sync), and the
		// client sends another request down the same conn, we can
		// actually handle them out of order because group state is
		// managed independently in its own loop. To ensure
		// serialization, we capture out of order responses and only
		// send them once the prior requests are replied to.
		//
		// (this is also why there is a seq in the clientReq)
		oooresp = make(map[uint32]clientResp)
	)
	for {
		resp, ok := oooresp[seq]
		if !ok {
			select {
			case resp = <-cc.respCh:
				if resp.seq != seq {
					oooresp[resp.seq] = resp
					continue
				}
				seq = resp.seq + 1
			case <-cc.done:
				return
			case <-cc.c.die:
				return
			}
		} else {
			delete(oooresp, seq)
			seq++
		}
		if err := resp.err; err != nil {
			cc.c.cfg.logger.Logf(LogLevelInfo, "client %s request unable to be handled: %v", who, err)
			return
		}

		buf = append(buf[:0], 0, 0, 0, 0, 0, 0, 0, 0) // size (4) + correlation ID (4)
		if resp.kresp.IsFlexible() && resp.kresp.Key() != 18 {
			buf = append(buf, 0) // empty tagged fields section
		}
		buf = resp.kresp.AppendTo(buf)

		binary.BigEndian.PutUint32(buf[:4], uint32(len(buf)-4))
		binary.BigEndian.PutUint32(buf[4:8], uint32(resp.corr))

		go func() {
			_, err := cc.conn.Write(buf)
			writeCh <- err
		}()

		var err error
		select {
		case <-cc.c.die:
			return
		case err = <-writeCh:
		}
		if err != nil {
			return
		}
	}
}
