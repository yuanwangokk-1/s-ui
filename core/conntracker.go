package core

import (
	"context"
	"net"
	"s-ui/database/model"
	"sync"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common/atomic"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/network"
)

type Counter struct {
	read  *atomic.Int64
	write *atomic.Int64
}

type ConnTracker struct {
	access    sync.Mutex
	createdAt time.Time
	inbounds  map[string]Counter
	outbounds map[string]Counter
	users     map[string]Counter
}

func NewConnTracker() *ConnTracker {
	return &ConnTracker{
		createdAt: time.Now(),
		inbounds:  make(map[string]Counter),
		outbounds: make(map[string]Counter),
		users:     make(map[string]Counter),
	}
}

func (c *ConnTracker) getReadCounters(inbound string, outbound string, user string) ([]*atomic.Int64, []*atomic.Int64) {
	var readCounter []*atomic.Int64
	var writeCounter []*atomic.Int64
	c.access.Lock()
	if inbound != "" {
		readCounter = append(readCounter, c.loadOrCreateCounter(&c.inbounds, inbound).read)
		writeCounter = append(writeCounter, c.inbounds[inbound].write)
	}
	if outbound != "" {
		readCounter = append(readCounter, c.loadOrCreateCounter(&c.outbounds, outbound).read)
		writeCounter = append(writeCounter, c.outbounds[outbound].write)
	}
	if user != "" {
		readCounter = append(readCounter, c.loadOrCreateCounter(&c.users, user).read)
		writeCounter = append(writeCounter, c.users[user].write)
	}
	c.access.Unlock()
	return readCounter, writeCounter
}

func (c *ConnTracker) loadOrCreateCounter(obj *map[string]Counter, name string) Counter {
	counter, loaded := (*obj)[name]
	if loaded {
		return counter
	}
	counter = Counter{read: &atomic.Int64{}, write: &atomic.Int64{}}
	(*obj)[name] = counter
	return counter
}

func (c *ConnTracker) RoutedConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) net.Conn {
	readCounter, writeCounter := c.getReadCounters(metadata.Inbound, matchOutbound.Tag(), metadata.User)
	return bufio.NewInt64CounterConn(conn, readCounter, writeCounter)
}

func (c *ConnTracker) RoutedPacketConnection(ctx context.Context, conn network.PacketConn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) network.PacketConn {
	readCounter, writeCounter := c.getReadCounters(metadata.Inbound, matchOutbound.Tag(), metadata.User)
	return bufio.NewInt64CounterPacketConn(conn, readCounter, writeCounter)
}

func (c *ConnTracker) GetStats() *[]model.Stats {
	c.access.Lock()
	defer c.access.Unlock()

	dt := time.Now().Unix()

	s := []model.Stats{}
	for inbound, counter := range c.inbounds {
		down := counter.write.Swap(0)
		up := counter.read.Swap(0)
		if down > 0 || up > 0 {
			s = append(s, model.Stats{
				DateTime:  dt,
				Resource:  "inbound",
				Tag:       inbound,
				Direction: false,
				Traffic:   down,
			}, model.Stats{
				DateTime:  dt,
				Resource:  "inbound",
				Tag:       inbound,
				Direction: true,
				Traffic:   up,
			})
		}
	}

	for outbound, counter := range c.outbounds {
		down := counter.write.Swap(0)
		up := counter.read.Swap(0)
		if down > 0 || up > 0 {
			s = append(s, model.Stats{
				DateTime:  dt,
				Resource:  "outbound",
				Tag:       outbound,
				Direction: false,
				Traffic:   down,
			}, model.Stats{
				DateTime:  dt,
				Resource:  "outbound",
				Tag:       outbound,
				Direction: true,
				Traffic:   up,
			})
		}
	}

	for user, counter := range c.users {
		down := counter.write.Swap(0)
		up := counter.read.Swap(0)
		if down > 0 || up > 0 {
			s = append(s, model.Stats{
				DateTime:  dt,
				Resource:  "user",
				Tag:       user,
				Direction: false,
				Traffic:   down,
			}, model.Stats{
				DateTime:  dt,
				Resource:  "user",
				Tag:       user,
				Direction: true,
				Traffic:   up,
			})
		}
	}
	return &s
}
