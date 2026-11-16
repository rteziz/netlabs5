package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const limit = 64 * 1024

type phase int

const (
	greeting phase = iota
	request
	resolving
	connecting
	relay
	closing
)

type client struct {
	fd, peer, port               int
	state                        phase
	up, down                     []byte
	eof, peerEOF, shut, peerShut bool
	id                           uint16
}
type server struct {
	listener, dns int
	clients       map[int]*client
	queries       map[uint16]*query
	scratch       [limit]byte
}

func socket(kind int) (int, error) {
	fd, e := unix.Socket(unix.AF_INET, kind, 0)
	if e != nil {
		return -1, e
	}
	if e = unix.SetNonblock(fd, true); e != nil {
		unix.Close(fd)
		return -1, e
	}
	return fd, nil
}
func transient(e error) bool { return e == unix.EAGAIN || e == unix.EINTR }
func (s *server) drop(c *client) {
	if _, ok := s.clients[c.fd]; !ok {
		return
	}
	if q := s.queries[c.id]; q != nil && q.c == c {
		delete(s.queries, c.id)
	}
	unix.Close(c.fd)
	if c.peer >= 0 {
		unix.Close(c.peer)
	}
	delete(s.clients, c.fd)
}
func (c *client) fail(code byte) {
	c.down = append(c.down, 5, code, 0, 1, 0, 0, 0, 0, 0, 0)
	c.state = closing
}
func (c *client) connect(ip [4]byte) {
	fd, e := socket(unix.SOCK_STREAM)
	if e != nil {
		c.fail(1)
		return
	}
	c.peer = fd
	e = unix.Connect(fd, &unix.SockaddrInet4{Addr: ip, Port: c.port})
	c.state = connecting
	if e == nil {
		c.connected()
	} else if e != unix.EINPROGRESS {
		c.connectError(e)
	}
}
func (c *client) connectError(e error) {
	code := byte(1)
	switch e {
	case unix.ECONNREFUSED:
		code = 5
	case unix.ENETUNREACH:
		code = 3
	case unix.EHOSTUNREACH:
		code = 4
	}
	c.fail(code)
}
func (c *client) connected() {
	a, e := unix.Getsockname(c.peer)
	if e != nil {
		c.fail(1)
		return
	}
	v := a.(*unix.SockaddrInet4)
	c.down = append(c.down, 5, 0, 0, 1)
	c.down = append(c.down, v.Addr[:]...)
	c.down = append(c.down, byte(v.Port>>8), byte(v.Port))
	c.state = relay
}
func (s *server) parse(c *client) {
	b := c.up
	if c.state == greeting {
		if len(b) < 2 {
			return
		}
		n := 2 + int(b[1])
		if len(b) < n {
			return
		}
		if b[0] != 5 || !bytes.Contains(b[2:n], []byte{0}) {
			c.down = append(c.down, 5, 255)
			c.state = closing
			return
		}
		c.down = append(c.down, 5, 0)
		c.state = request
		b = b[n:]
		c.up = b
	}
	if c.state != request || len(b) < 4 {
		return
	}
	if b[0] != 5 || b[2] != 0 {
		c.fail(1)
		return
	}
	if b[1] != 1 {
		c.fail(7)
		return
	}
	n := 10
	switch b[3] {
	case 1:
	case 3:
		if len(b) < 5 {
			return
		}
		n = 7 + int(b[4])
	default:
		c.fail(8)
		return
	}
	if len(b) < n {
		return
	}
	c.port = int(b[n-2])*256 + int(b[n-1])
	c.up = b[n:]
	if b[3] == 1 {
		c.connect([4]byte(b[4:8]))
	} else {
		s.resolve(c, string(b[5:n-2]))
	}
}
func (s *server) io(c *client, peer bool, events int16) {
	fd := c.fd
	in, out := &c.up, &c.down
	end := &c.eof
	if peer {
		fd = c.peer
		in, out = &c.down, &c.up
		end = &c.peerEOF
		if c.state == connecting {
			errno, e := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_ERROR)
			if e != nil {
				c.fail(1)
			} else if errno != 0 {
				c.connectError(unix.Errno(errno))
			} else {
				c.connected()
			}
			return
		}
	}
	if events&unix.POLLOUT != 0 && len(*out) > 0 {
		n, e := unix.Write(fd, *out)
		if e != nil && !transient(e) {
			s.drop(c)
			return
		}
		if n > 0 {
			*out = (*out)[n:]
		}
	}
	if events&(unix.POLLIN|unix.POLLHUP) != 0 && !*end && c.state != closing && len(*in) < limit {
		n, e := unix.Read(fd, s.scratch[:limit-len(*in)])
		if e != nil && !transient(e) {
			s.drop(c)
			return
		}
		if n > 0 {
			*in = append(*in, s.scratch[:n]...)
			if !peer {
				s.parse(c)
			}
		} else if e == nil {
			*end = true
		}
	}
	if events&unix.POLLERR != 0 {
		s.drop(c)
		return
	}
	if c.eof && c.state <= request {
		c.state = closing
	}
	if c.state == closing && len(c.down) == 0 {
		s.drop(c)
		return
	}
	if c.state == relay {
		if c.eof && len(c.up) == 0 && !c.peerShut {
			unix.Shutdown(c.peer, unix.SHUT_WR)
			c.peerShut = true
		}
		if c.peerEOF && len(c.down) == 0 && !c.shut {
			unix.Shutdown(c.fd, unix.SHUT_WR)
			c.shut = true
		}
		if c.shut && c.peerShut {
			s.drop(c)
		}
	}
}
func interest(read, write bool) int16 {
	var v int16
	if read {
		v |= unix.POLLIN
	}
	if write {
		v |= unix.POLLOUT
	}
	return v
}
func (s *server) loop() error {
	type endpoint struct {
		c    *client
		peer bool
	}
	for {
		timeout, wantWrite := s.timers(time.Now())
		fds := []unix.PollFd{{Fd: int32(s.listener), Events: unix.POLLIN}, {Fd: int32(s.dns), Events: interest(true, wantWrite)}}
		ends := []endpoint{}
		add := func(fd int, c *client, peer bool, mask int16) {
			if mask != 0 {
				fds = append(fds, unix.PollFd{Fd: int32(fd), Events: mask})
				ends = append(ends, endpoint{c, peer})
			}
		}
		for _, c := range s.clients {
			add(c.fd, c, false, interest(!c.eof && c.state != closing && len(c.up) < limit, len(c.down) > 0))
			add(c.peer, c, true, interest(c.state == relay && !c.peerEOF && len(c.down) < limit, c.state == connecting || c.state == relay && len(c.up) > 0))
		}
		_, e := unix.Poll(fds, timeout)
		if e == unix.EINTR {
			continue
		}
		if e != nil {
			return e
		}
		if fds[0].Revents != 0 {
			fd, _, e := unix.Accept(s.listener)
			if e == nil {
				if unix.SetNonblock(fd, true) == nil {
					s.clients[fd] = &client{fd: fd, peer: -1}
				} else {
					unix.Close(fd)
				}
			} else if !transient(e) && e != unix.ECONNABORTED {
				return e
			}
		}
		if fds[1].Revents&(unix.POLLIN|unix.POLLERR) != 0 {
			s.answer()
		}
		if fds[1].Revents&unix.POLLOUT != 0 {
			s.sendQueries()
		}
		for i, p := range ends {
			if ev := fds[i+2].Revents; ev != 0 && s.clients[p.c.fd] == p.c {
				s.io(p.c, p.peer, ev)
			}
		}
	}
}
func run() error {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		return errors.New("usage: lab5 PORT [DNS_IP:PORT]")
	}
	port, e := strconv.Atoi(os.Args[1])
	if e != nil || port < 1 || port > 65535 {
		return errors.New("invalid port")
	}
	var resolver netip.AddrPort
	if len(os.Args) == 3 {
		resolver, e = netip.ParseAddrPort(os.Args[2])
	} else {
		data, err := os.ReadFile("/etc/resolv.conf")
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(data), "\n") {
			f := strings.Fields(line)
			if len(f) >= 2 && f[0] == "nameserver" {
				ip, err := netip.ParseAddr(f[1])
				if err == nil && ip.Is4() {
					resolver = netip.AddrPortFrom(ip, 53)
					break
				}
			}
		}
	}
	if e != nil || !resolver.Addr().Is4() || resolver.Port() == 0 {
		return errors.New("expected IPv4 DNS resolver with port")
	}
	s := server{clients: make(map[int]*client), queries: make(map[uint16]*query)}
	s.listener, e = socket(unix.SOCK_STREAM)
	if e != nil {
		return e
	}
	defer unix.Close(s.listener)
	unix.SetsockoptInt(s.listener, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if e = unix.Bind(s.listener, &unix.SockaddrInet4{Port: port}); e != nil {
		return e
	}
	if e = unix.Listen(s.listener, 128); e != nil {
		return e
	}
	s.dns, e = socket(unix.SOCK_DGRAM)
	if e != nil {
		return e
	}
	defer unix.Close(s.dns)
	if e = unix.Connect(s.dns, &unix.SockaddrInet4{Addr: resolver.Addr().As4(), Port: int(resolver.Port())}); e != nil {
		return e
	}
	return s.loop()
}
func main() {
	runtime.LockOSThread()
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
