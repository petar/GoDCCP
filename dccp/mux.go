// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
	"os"
	"sync"
	"time"
)

// XXX: Every other Label is 00
// XXX: When everyone looses ptr to mux, the object remains in memory since loop() is keeping it
// XXX: mux should not pass "cargo" bigger than allowed size. flow.Read() should fail if provided
//      buffer cannot accommodate them.
// XXX: Add logic to handle fragmentation (length field, catch errors)

// TODO: Keep track of flows with only one packet (likely caused by fragmentation)

// mux{} is a thin protocol layer that works on top of a connection-less packet layer, like UDP.
// mux{} multiplexes packets into flows. A flow is a point-to-point connection, which has no
// congestion or reliability mechanism.
//
// mux{} implements a mechanism for dropping packets that linger (are received) for up to
// one minute after their respective flow has been closed.
//
// mux{} force-closes flows that have experienced no activity for 10 mins
type mux struct {
	sync.Mutex
	link           Link
	fragLen        int
	flowsLocal     map[uint64]*flow  // Active flows hashed by local label
	flowsRemote    map[uint64]*flow
	lingerLocal    map[uint64]int64  // Local labels of recently-closed flows mapped to time of closure
	lingerRemote   map[uint64]int64
	acceptChan     chan *flow
}
const (
       	muxLingerTime = 60e9  // 1 min in nanoseconds
	muxExpireTime = 600e9 // 10 min in nanoseconds
)

// muxHeader{} is an internal data structure that carries a parsed switch packet,
// which contains a flow id and a generic DCCP header
type muxHeader struct {
	Msg    *muxMsg
	Cargo  []byte
}

func newMux(link Link, fragLen int) *mux {
	m := &mux{ 
		link:         link, 
		fragLen:      fragLen,
		flowsLocal:   make(map[uint64]*flow),
		flowsRemote:  make(map[uint64]*flow),
		lingerLocal:  make(map[uint64]int64),
		lingerRemote: make(map[uint64]int64),
		acceptChan:   make(chan *flow),
	}
	go m.readLoop()
	go m.expireLingeringLoop()
	go m.expireLoop()
	return m
}

const fragSafety = 5

func (m *mux) readLoop() {
	for {
		// Check that mux is still open
		m.Lock()
		link := m.link
		m.Unlock()
		if link == nil {
			break
		}

		// Read incoming packet
		buf := make([]byte, m.fragLen + fragSafety)
		n, addr, err := link.ReadFrom(buf)
		if err != nil {
			break
		}

		// Check that it is not oversized
		if len(buf)-n < fragSafety {
			break
		}

		// Read mux header
		msg, cargo, err := readMuxHeader(buf[:n])
		if err != nil {
			continue
		}

		m.process(msg, cargo, addr)
	}
	close(m.acceptChan)
	m.Lock()
	for _, f := range m.flowsLocal {
		f.foreclose()
	}
	for _, f := range m.flowsRemote {
		f.foreclose()
	}
	m.Unlock()
}

func (m *mux) process(msg *muxMsg, cargo []byte, addr net.Addr) {
	// REMARK: By design, only one copy of process() can run at a time (*)

	// Every packet must have a source (remote) label
	if msg.Source == nil {
		return
	}

	// If it's a lingering packet, drop it
	if m.isLingering(msg.Sink, msg.Source) {
		return
	}

	var f *flow
	// Does the packet have a sink label?
	if msg.Sink != nil {
		// If yes, then we must have a matching flow
		f = m.findLocal(msg.Sink)
		if f == nil {
			return
		}
		// Check if this is the first time we hear about the remote label on this flow
		if f.getRemote() == nil {
			// If yes, then we just discovered the remote label. Save it.
			// Remark (*), above, ensures that the next 4 lines are executed atomically
			f.setRemote(msg.Source)
			m.Lock()
			m.flowsRemote[msg.Source.Hash()] = f
			m.Unlock()
		} else if !f.getRemote().Equal(msg.Source) {
			return
		}
	} else {
		f = m.findRemote(msg.Source)
		if f == nil {
			f = m.accept(msg.Source, addr)
		}
	}

	f.ch <- muxHeader{msg, cargo}
}

func (m *mux) accept(remote *Label, addr net.Addr) *flow {
	if remote == nil {
		panic("remote == nil")
	}

	ch := make(chan muxHeader)
	local := ChooseLabel()
	f := newFlow(addr, m, ch, local, remote)

	m.Lock()
	m.flowsLocal[local.Hash()] = f
	m.flowsRemote[remote.Hash()] = f
	m.Unlock()

	m.acceptChan <- f

	return f
}

// readMuxHeader() reads a header consisting of mux-specific flow information followed by data
func readMuxHeader(p []byte) (msg *muxMsg, cargo []byte, err os.Error) {
	msg, n, err := readMuxMsg(p)
	if err != nil {
		return nil, nil, err
	}
	cargo = p[n:]
	return msg, cargo, nil
}

// Accept() returns the first incoming flow request
func (m *mux) Accept() (c net.Conn, err os.Error) {
	f, ok := <-m.acceptChan
	if !ok {
		return nil, os.EBADF
	}
	return f, nil
}

// findLocal() checks if there already exists a flow corresponding to the given local label
func (m *mux) findLocal(local *Label) *flow {
	if local == nil {
		return nil
	}
	m.Lock()
	defer m.Unlock()

	return m.flowsLocal[local.Hash()]
}

// findRemote() checks if there already exists a flow corresponding to the given remote label
func (m *mux) findRemote(remote *Label) *flow {
	if remote == nil {
		return nil
	}
	m.Lock()
	defer m.Unlock()

	return m.flowsRemote[remote.Hash()]
}

func (m *mux) Dial(addr net.Addr) (c net.Conn, err os.Error) {
	ch := make(chan muxHeader)
	local := ChooseLabel()
	f := newFlow(addr, m, ch, local, nil)

	m.Lock()
	m.flowsLocal[local.Hash()] = f
	m.Unlock()

	return f, nil
}

// expireLoop() force-closes flows that have been inactive for more than 10 min
func (m *mux) expireLoop() {
	for {
		time.Sleep(muxExpireTime)

		// Check if mux has been closed
		m.Lock()
		link := m.link
		m.Unlock()
		if link == nil {
			break
		}

		now := m.Now()
		m.Lock()
		// All active flows have local labels, so it's enough to iterate just flowsLocal[]
		for _, f := range m.flowsLocal {
			if now-f.LastWrite() > muxExpireTime {
				f.foreclose()
			}
		}
		m.Unlock()
	}
}

// isLingering() returns true if the labels of this packet pertain to a flow
// that has been closed in the past minute.
func (m *mux) isLingering(local, remote *Label) bool {
	m.Lock()
	defer m.Unlock()

	if local != nil {
		if _, yes := m.lingerLocal[local.Hash()]; yes {
			return true
		}
	}
	if remote != nil {
		if _, yes := m.lingerRemote[remote.Hash()]; yes {
			return true
		}
	}
	return false
}

// expireLingeringLoop() removes the labels of connections that have been closed for more than
// a minute from the data structure that remembers them
func (m *mux) expireLingeringLoop() {
	for {
		time.Sleep(muxLingerTime)

		// Check if mux has been closed
		m.Lock()
		link := m.link
		m.Unlock()
		if link == nil {
			break
		}

		now := m.Now()
		m.Lock()
		for h, t := range m.lingerLocal {
			if now - t >= muxLingerTime {
				m.lingerLocal[h] = 0, false
			}
		}
		for h, t := range m.lingerRemote {
			if now - t >= muxLingerTime {
				m.lingerRemote[h] = 0, false
			}
		}
		m.Unlock()
	}
}

// del() removes the flow with the specified labels from the data structure, if it still exists
func (m *mux) del(local *Label, remote *Label) {
	m.Lock()
	defer m.Unlock()

	now := m.Now()
	if local != nil {
		m.flowsLocal[local.Hash()] = nil, false
		if _, alreadyClosed := m.lingerLocal[local.Hash()]; !alreadyClosed {
			m.lingerLocal[local.Hash()] = now
		}
	}
	if remote != nil {
		m.flowsRemote[remote.Hash()] = nil, false
		if _, alreadyClosed := m.lingerRemote[remote.Hash()]; !alreadyClosed {
			m.lingerRemote[remote.Hash()] = now
		}
	}
}

func (m *mux) write(msg *muxMsg, cargo []byte, addr net.Addr) (n int, err os.Error) {
	if msg.Len() + len(cargo) > m.fragLen {
		return 0, os.NewError("fragment too big")
	}
	m.Lock()
	link := m.link
	m.Unlock()
	if link == nil {
		return 0, os.EBADF
	}

	buf := make([]byte, msg.Len() + len(cargo))
	msg.Write(buf)
	copy(buf[msg.Len():], cargo)
	
	n, err = link.WriteTo(buf, addr)
	return max(0, n-msg.Len()), err
}

func max(i,j int) int {
	if i > j {
		return i
	}
	return j
}

// Close() closes the mux and signals all outstanding connections
// that it is time to terminate
func (m *mux) Close() os.Error {
	m.Lock()
	link := m.link
	m.link = nil
	for _, f := range m.flowsLocal {
		f.foreclose()
	}
	for _, f := range m.flowsRemote {
		f.foreclose()
	}
	m.Unlock()
	if link == nil {
		return os.EBADF
	}
	return link.Close()
}

// Now() returns the current time in nanoseconds
func (m *mux) Now() int64 { return time.Nanoseconds() }
