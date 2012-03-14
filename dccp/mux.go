// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
	"syscall"
	"time"
)

// Mux is a thin protocol layer that works on top of a connection-less packet layer, like UDP.
// Mux multiplexes packets into flows. A flow is a point-to-point connection, which has no
// congestion or reliability mechanism.
//
// Mux implements a mechanism for dropping packets that linger (are received) for up to
// one minute after their respective flow has been closed.
//
// Mux force-closes flows that have experienced no activity for 10 mins
type Mux struct {
	Mutex
	link         Link
	flowsLocal   map[uint64]*flow // Active flows hashed by local label
	flowsRemote  map[uint64]*flow
	lingerLocal  map[uint64]time.Time // Local labels of recently-closed flows mapped to time of closure
	lingerRemote map[uint64]time.Time
	acceptChan   chan *flow
}

const (
	MuxLingerTime = 60e9  // 1 min in nanoseconds
	MuxExpireTime = 600e9 // 10 min in nanoseconds
	MuxReadSafety = 5
)

// muxHeader is an internal data structure that carries a parsed switch packet,
// which contains a flow ID and a DCCP header
type muxHeader struct {
	Msg   *muxMsg
	Cargo []byte
}

// NewMux creates a new Mux object, using the connection-less packet interface link
func NewMux(link Link) *Mux {
	m := &Mux{
		link:         link,
		flowsLocal:   make(map[uint64]*flow),
		flowsRemote:  make(map[uint64]*flow),
		lingerLocal:  make(map[uint64]time.Time),
		lingerRemote: make(map[uint64]time.Time),
		acceptChan:   make(chan *flow),
	}
	go m.readLoop()
	go m.expireLingeringLoop()
	go m.expireLoop()
	return m
}

// Accept() returns the first incoming flow request
func (m *Mux) Accept() (c SegmentConn, err error) {
	f, ok := <-m.acceptChan
	if !ok {
		return nil, syscall.EBADF
	}
	return f, nil
}

// Dial opens a packet-based connection to the Link-layer addr
func (m *Mux) Dial(addr net.Addr) (c SegmentConn, err error) {
	ch := make(chan muxHeader)
	local := ChooseLabel()
	f := newFlow(addr, m, ch, m.cargoMaxLen(), local, nil)

	m.Lock()
	m.flowsLocal[local.Hash()] = f
	m.Unlock()

	return f, nil
}

// Close() closes the mux and signals all outstanding connections
// that it is time to terminate
func (m *Mux) Close() error {
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
		return syscall.EBADF
	}
	return link.Close()
}

func (m *Mux) readLoop() {
	for {
		// Check that mux is still open
		m.Lock()
		link := m.link
		m.Unlock()
		if link == nil {
			break
		}

		// Read incoming packet
		buf := make([]byte, m.link.GetMTU()+MuxReadSafety)
		n, addr, err := link.ReadFrom(buf)
		if err != nil {
			break
		}

		// Check that packet is not oversized
		if len(buf)-n < MuxReadSafety {
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

func (m *Mux) process(msg *muxMsg, cargo []byte, addr net.Addr) {
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

func (m *Mux) accept(remote *Label, addr net.Addr) *flow {
	if remote == nil {
		panic("remote == nil")
	}

	ch := make(chan muxHeader)
	local := ChooseLabel()
	f := newFlow(addr, m, ch, m.cargoMaxLen(), local, remote)

	m.Lock()
	m.flowsLocal[local.Hash()] = f
	m.flowsRemote[remote.Hash()] = f
	m.Unlock()

	m.acceptChan <- f

	return f
}

// readMuxHeader() reads a header consisting of mux-specific flow information followed by data
func readMuxHeader(p []byte) (msg *muxMsg, cargo []byte, err error) {
	msg, n, err := readMuxMsg(p)
	if err != nil {
		return nil, nil, err
	}
	cargo = p[n:]
	return msg, cargo, nil
}

// findLocal() checks if there already exists a flow corresponding to the given local label
func (m *Mux) findLocal(local *Label) *flow {
	if local == nil {
		return nil
	}
	m.Lock()
	defer m.Unlock()

	return m.flowsLocal[local.Hash()]
}

// findRemote() checks if there already exists a flow corresponding to the given remote label
func (m *Mux) findRemote(remote *Label) *flow {
	if remote == nil {
		return nil
	}
	m.Lock()
	defer m.Unlock()

	return m.flowsRemote[remote.Hash()]
}

// expireLoop() force-closes flows that have been inactive for more than 10 min
func (m *Mux) expireLoop() {
	for {
		time.Sleep(MuxExpireTime)

		// Check if mux has been closed
		m.Lock()
		link := m.link
		m.Unlock()
		if link == nil {
			break
		}

		now := time.Now()
		m.Lock()
		// All active flows have local labels, so it's enough to iterate just flowsLocal[]
		for _, f := range m.flowsLocal {
			if now.Sub(f.LastWriteTime()) > MuxExpireTime {
				f.foreclose()
			}
		}
		m.Unlock()
	}
}

// isLingering() returns true if the labels of this packet pertain to a flow
// that has been closed in the past minute.
func (m *Mux) isLingering(local, remote *Label) bool {
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
func (m *Mux) expireLingeringLoop() {
	for {
		time.Sleep(MuxLingerTime)

		// Check if mux has been closed
		m.Lock()
		link := m.link
		m.Unlock()
		if link == nil {
			break
		}

		now := time.Now()
		m.Lock()
		for h, t := range m.lingerLocal {
			if now.Sub(t) >= MuxLingerTime {
				delete(m.lingerLocal, h)
			}
		}
		for h, t := range m.lingerRemote {
			if now.Sub(t) >= MuxLingerTime {
				delete(m.lingerRemote, h)
			}
		}
		m.Unlock()
	}
}

// del() removes the flow with the specified labels from the data structure, if it still exists
func (m *Mux) del(local *Label, remote *Label) {
	m.Lock()
	defer m.Unlock()

	now := time.Now()
	if local != nil {
		delete(m.flowsLocal, local.Hash())
		if _, alreadyClosed := m.lingerLocal[local.Hash()]; !alreadyClosed {
			m.lingerLocal[local.Hash()] = now
		}
	}
	if remote != nil {
		delete(m.flowsRemote, remote.Hash())
		if _, alreadyClosed := m.lingerRemote[remote.Hash()]; !alreadyClosed {
			m.lingerRemote[remote.Hash()] = now
		}
	}
}

func (m *Mux) cargoMaxLen() int { return m.link.GetMTU() - muxMsgFootprint }

func (m *Mux) write(msg *muxMsg, block []byte, addr net.Addr) error {
	m.Lock()
	link := m.link
	m.Unlock()
	if link == nil {
		return syscall.EBADF
	}

	buf := make([]byte, muxMsgFootprint+len(block))
	msg.Write(buf)
	copy(buf[muxMsgFootprint:], block)

	n, err := link.WriteTo(buf, addr)
	if n != muxMsgFootprint+len(block) {
		panic("block divided")
	}
	return err
}
