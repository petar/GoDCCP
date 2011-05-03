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

// XXX: Drop packets going to recently closed flows. Combine with
// XXX: Keepalive mechanism (no, this belongs in DCCP), just close connections that are idle for a while
// XXX: Add logic to handle fragmentation (length field, catch errors)
// XXX: When everyone looses ptr to mux, the object remains in memory since loop() is keeping it
// XXX: Every other Label is 00
// XXX: mux should not pass "cargo" bigger than allowed size. flow.Read() should fail if provided
//      buffer cannot accommodate them.

// TODO: Keep track of flows with only one packet (likely caused by fragmentation)

// mux{} helps multiplex the connection-less physical packets among 
// multiple logical connections based on their logical flow ID.
type mux struct {
	sync.Mutex
	link        Link
	fragLen     int
	flowsLocal  map[uint64]*flow  // Flows hashed by local label
	flowsRemote map[uint64]*flow  // Flows hashed by remote label
	acceptChan  chan *flow
}

// muxHeader{} is an internal data structure that carries a parsed switch packet,
// which contains a flow id and a generic DCCP header
type muxHeader struct {
	Msg    *muxMsg
	Cargo  []byte
}

func newMux(link Link, fragLen int) *mux {
	m := &mux{ 
		link:        link, 
		fragLen:     fragLen,
		flowsLocal:  make(map[uint64]*flow),
		flowsRemote: make(map[uint64]*flow),
		acceptChan:  make(chan *flow),
	}
	go m.loop()
	return m
}

const fragSafety = 5

func (m *mux) loop() {
	for {
		m.Lock()
		link := m.link
		m.Unlock()
		if link == nil {
			break
		}
		buf := make([]byte, m.fragLen + fragSafety)
		n, addr, err := link.ReadFrom(buf)
		if err != nil {
			break
		}
		if len(buf)-n < fragSafety {
			break
		}
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

// del() removes the flow with the specified labels from the data structure, if it still exists
func (m *mux) del(local *Label, remote *Label) {
	m.Lock()
	defer m.Unlock()

	if local != nil {
		m.flowsLocal[local.Hash()] = nil, false
	}
	if remote != nil {
		m.flowsRemote[remote.Hash()] = nil, false
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
