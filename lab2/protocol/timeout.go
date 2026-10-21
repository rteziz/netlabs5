package protocol

import (
	"net"
	"time"
)

const (
	ConnectTimeout = 10 * time.Second
	IdleTimeout    = 30 * time.Second
)

type IdleConn struct {
	net.Conn
	Timeout time.Duration
}

func (c *IdleConn) Read(p []byte) (int, error) {
	if err := c.SetReadDeadline(time.Now().Add(c.Timeout)); err != nil {
		return 0, err
	}
	return c.Conn.Read(p)
}

func (c *IdleConn) Write(p []byte) (int, error) {
	if err := c.SetWriteDeadline(time.Now().Add(c.Timeout)); err != nil {
		return 0, err
	}
	return c.Conn.Write(p)
}
