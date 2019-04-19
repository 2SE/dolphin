package cluster

import (
	"bufio"
	"encoding/gob"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/rpc"
	"time"
)

var (
	defaultTimeout = time.Minute
)

func timeoutCodec(f func(interface{}) error, e interface{}, msg string) error {
	echan := make(chan error, 1)
	timeout := defaultTimeout
	if peerConnCnf != nil && peerConnCnf.ReqWaitAfter.Get() > 0 {
		timeout = peerConnCnf.ReqWaitAfter.Get()
	}

	go func() { echan <- f(e) }()
	select {
	case e := <-echan:
		return e
	case <-time.After(timeout):
		return fmt.Errorf("timeout %s", msg)
	}
}

type gobClientCodec struct {
	rwc    io.ReadWriteCloser
	dec    *gob.Decoder
	enc    *gob.Encoder
	encBuf *bufio.Writer
}

func (c *gobClientCodec) WriteRequest(r *rpc.Request, body interface{}) (err error) {
	if err = timeoutCodec(c.enc.Encode, r, "client write request"); err != nil {
		return
	}

	if err = timeoutCodec(c.enc.Encode, body, "client write request body"); err != nil {
		return
	}

	return c.encBuf.Flush()
}

func (c *gobClientCodec) ReadResponseHeader(r *rpc.Response) error {
	return c.dec.Decode(r)
}

func (c *gobClientCodec) ReadResponseBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *gobClientCodec) Close() error {
	return c.rwc.Close()
}

type gobServerCodec struct {
	rwc    io.ReadWriteCloser
	dec    *gob.Decoder
	enc    *gob.Encoder
	encBuf *bufio.Writer
	closed bool
}

func (c *gobServerCodec) ReadRequestHeader(r *rpc.Request) error {
	return timeoutCodec(c.dec.Decode, r, "server read request header")
}

func (c *gobServerCodec) ReadRequestBody(body interface{}) error {
	return timeoutCodec(c.dec.Decode, body, "server read request body")
}

func (c *gobServerCodec) WriteResponse(r *rpc.Response, body interface{}) (err error) {
	if err = timeoutCodec(c.enc.Encode, r, "server write response"); err != nil {
		if c.encBuf.Flush() == nil {
			log.Println("rpc: gob error encoding response:", err)
			c.Close()
		}
		return
	}

	if err = timeoutCodec(c.enc.Encode, body, "server write response body"); err != nil {
		if c.encBuf.Flush() == nil {
			log.Println("rpc: gob error encoding body:", err)
			c.Close()
		}
		return
	}
	return c.encBuf.Flush()
}

func (c *gobServerCodec) Close() error {
	if c.closed {
		// Only call c.rwc.Close once; otherwise the semantics are undefined.
		return nil
	}
	c.closed = true
	return c.rwc.Close()
}
