package main

import (
	"bytes"
	"encoding/binary"
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

const (
	socksVersion  byte = 0x05
	socksReserved byte = 0x00

	authNoAuthentication    byte = 0x00
	authNoAcceptableMethods byte = 0xFF

	commandConnect byte = 0x01

	addressIPv4   byte = 0x01
	addressDomain byte = 0x03

	replySucceeded          byte = 0x00
	replyGeneralFailure     byte = 0x01
	replyNetworkUnreachable byte = 0x03
	replyHostUnreachable    byte = 0x04
	replyConnectionRefused  byte = 0x05
	replyCommandUnsupported byte = 0x07
	replyAddressUnsupported byte = 0x08
)

// Позиции и размеры полей сообщений SOCKS5, в байтах.
const (
	versionOffset = 0

	greetingMethodCountOffset = 1
	greetingHeaderSize        = 2
	greetingMethodsOffset     = greetingHeaderSize

	requestCommandOffset     = 1
	requestReservedOffset    = 2
	requestAddressTypeOffset = 3
	requestHeaderSize        = 4
	requestAddressOffset     = requestHeaderSize

	ipv4AddressSize = 4
	portSize        = 2
	ipv4RequestSize = requestHeaderSize + ipv4AddressSize + portSize

	domainLengthOffset    = requestAddressOffset
	domainLengthSize      = 1
	domainNameOffset      = domainLengthOffset + domainLengthSize
	domainRequestBaseSize = domainNameOffset + portSize
)

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
		err := unix.Close(fd)
		if err != nil {
			return 0, err
		}
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
	err := unix.Close(c.fd)
	if err != nil {
		return
	}
	if c.peer >= 0 {
		err := unix.Close(c.peer)
		if err != nil {
			return
		}
	}
	delete(s.clients, c.fd)
}
func (c *client) fail(code byte) {
	c.down = append(c.down, socksVersion, code, socksReserved, addressIPv4, 0, 0, 0, 0, 0, 0)
	c.state = closing
}
func (c *client) connect(ip [ipv4AddressSize]byte) {
	fd, e := socket(unix.SOCK_STREAM)
	if e != nil {
		c.fail(replyGeneralFailure)
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
	code := replyGeneralFailure
	switch e {
	case unix.ECONNREFUSED:
		code = replyConnectionRefused
	case unix.ENETUNREACH:
		code = replyNetworkUnreachable
	case unix.EHOSTUNREACH:
		code = replyHostUnreachable
	}
	c.fail(code)
}
func (c *client) connected() {
	a, e := unix.Getsockname(c.peer)
	if e != nil {
		c.fail(replyGeneralFailure)
		return
	}
	v := a.(*unix.SockaddrInet4)
	c.down = append(c.down, socksVersion, replySucceeded, socksReserved, addressIPv4)
	c.down = append(c.down, v.Addr[:]...)
	c.down = binary.BigEndian.AppendUint16(c.down, uint16(v.Port))
	c.state = relay
}
func (s *server) parse(c *client) {
	b := c.up
	if c.state == greeting {
		if len(b) < greetingHeaderSize {
			return
		}
		n := greetingHeaderSize + int(b[greetingMethodCountOffset])
		if len(b) < n {
			return
		}
		if b[versionOffset] != socksVersion || !bytes.Contains(b[greetingMethodsOffset:n], []byte{authNoAuthentication}) {
			c.down = append(c.down, socksVersion, authNoAcceptableMethods)
			c.state = closing
			return
		}
		c.down = append(c.down, socksVersion, authNoAuthentication)
		c.state = request
		b = b[n:]
		c.up = b
	}
	if c.state != request || len(b) < requestHeaderSize {
		return
	}
	if b[versionOffset] != socksVersion || b[requestReservedOffset] != socksReserved {
		c.fail(replyGeneralFailure)
		return
	}
	if b[requestCommandOffset] != commandConnect {
		c.fail(replyCommandUnsupported)
		return
	}
	n := ipv4RequestSize
	switch b[requestAddressTypeOffset] {
	case addressIPv4:
	case addressDomain:
		if len(b) < domainNameOffset {
			return
		}
		n = domainRequestBaseSize + int(b[domainLengthOffset])
	default:
		c.fail(replyAddressUnsupported)
		return
	}
	if len(b) < n {
		return
	}
	c.port = int(binary.BigEndian.Uint16(b[n-portSize : n]))
	c.up = b[n:]
	if b[requestAddressTypeOffset] == addressIPv4 {
		c.connect([ipv4AddressSize]byte(b[requestAddressOffset : requestAddressOffset+ipv4AddressSize]))
	} else {
		s.resolve(c, string(b[domainNameOffset:n-portSize]))
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
				c.fail(replyGeneralFailure)
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
			err := unix.Shutdown(c.peer, unix.SHUT_WR)
			if err != nil {
				return
			}
			c.peerShut = true
		}
		if c.peerEOF && len(c.down) == 0 && !c.shut {
			err := unix.Shutdown(c.fd, unix.SHUT_WR)
			if err != nil {
				return
			}
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
func serve() error {
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
		for line := range strings.SplitSeq(string(data), "\n") {
			f := strings.Fields(line)
			if len(f) >= 2 && f[0] == "nameserver" {
				ip, err := netip.ParseAddr(f[1])
				if err == nil && ip.Is4() {
					resolver = netip.AddrPortFrom(ip, defaultDNSPort)
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
	defer func(fd int) {
		err := unix.Close(fd)
		if err != nil {

		}
	}(s.listener)
	err := unix.SetsockoptInt(s.listener, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if err != nil {
		return err
	}
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
	defer func(fd int) {
		err := unix.Close(fd)
		if err != nil {

		}
	}(s.dns)
	if e = unix.Connect(s.dns, &unix.SockaddrInet4{Addr: resolver.Addr().As4(), Port: int(resolver.Port())}); e != nil {
		return e
	}
	return s.loop()
}
func run() {
	runtime.LockOSThread()
	if e := serve(); e != nil {
		_, _ = fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
