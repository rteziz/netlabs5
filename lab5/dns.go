package main

import (
	"math/rand/v2"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
	"golang.org/x/sys/unix"
)

const dnsTimeout = 2 * time.Second

type query struct {
	c        *client
	name     dnsmessage.Name
	packet   []byte
	attempts int
	pending  bool
	deadline time.Time
}

func (s *server) resolve(c *client, host string) {
	if host == "" || len(s.queries) == 65536 {
		c.fail(4)
		return
	}
	name, e := dnsmessage.NewName(strings.TrimSuffix(host, ".") + ".")
	if e != nil {
		c.fail(4)
		return
	}
	id := uint16(rand.Uint32())
	for s.queries[id] != nil {
		id++
	}
	msg := dnsmessage.Message{Header: dnsmessage.Header{ID: id, RecursionDesired: true}, Questions: []dnsmessage.Question{{Name: name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}}}
	packet, e := msg.Pack()
	if e != nil {
		c.fail(4)
		return
	}
	c.id = id
	c.state = resolving
	s.queries[id] = &query{c: c, name: name, packet: packet, pending: true, attempts: 1, deadline: time.Now().Add(dnsTimeout)}
}
func (s *server) timers(now time.Time) (int, bool) {
	wait := -1
	write := false
	for id, q := range s.queries {
		if !now.Before(q.deadline) {
			if q.attempts == 3 {
				q.c.fail(4)
				delete(s.queries, id)
				continue
			}
			q.attempts++
			q.pending = true
			q.deadline = now.Add(dnsTimeout)
		}
		ms := int((q.deadline.Sub(now) + time.Millisecond - 1) / time.Millisecond)
		if wait < 0 || ms < wait {
			wait = ms
		}
		write = write || q.pending
	}
	return wait, write
}
func (s *server) sendQueries() {
	for _, q := range s.queries {
		if q.pending {
			_, e := unix.Write(s.dns, q.packet)
			if transient(e) {
				return
			}
			q.pending = false
		}
	}
}
func (s *server) answer() {
	n, e := unix.Read(s.dns, s.scratch[:])
	if e != nil {
		return
	}
	var p dnsmessage.Parser
	h, e := p.Start(s.scratch[:n])
	if e != nil || !h.Response || h.OpCode != 0 {
		return
	}
	q := s.queries[h.ID]
	if q == nil {
		return
	}
	question, e := p.Question()
	if e != nil || question.Type != dnsmessage.TypeA || question.Class != dnsmessage.ClassINET || !strings.EqualFold(question.Name.String(), q.name.String()) {
		return
	}
	if _, e = p.Question(); e != dnsmessage.ErrSectionDone {
		return
	}
	delete(s.queries, h.ID)
	aliases := map[string]string{}
	ips := map[string][4]byte{}
	for h.RCode == dnsmessage.RCodeSuccess {
		r, e := p.Answer()
		if e != nil {
			break
		}
		if r.Header.Class != dnsmessage.ClassINET {
			continue
		}
		key := strings.ToLower(r.Header.Name.String())
		switch body := r.Body.(type) {
		case *dnsmessage.AResource:
			ips[key] = body.A
		case *dnsmessage.CNAMEResource:
			aliases[key] = strings.ToLower(body.CNAME.String())
		}
	}
	key := strings.ToLower(q.name.String())
	for key != "" {
		if ip, ok := ips[key]; ok {
			q.c.connect(ip)
			return
		}
		next := aliases[key]
		delete(aliases, key)
		key = next
	}
	q.c.fail(4)
}
