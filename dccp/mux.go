// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"log"
	"os"
	"sync"
)

// XXX: Drop packets going to recently closed flows. Combine with
// XXX: Keepalive mechanism (no, this belongs in DCCP), just close connections that are idle for a while

// mux{} helps multiplex the connection-less physical packets among 
// multiple logical connections based on their logical flow ID.
type mux struct {
	sync.Mutex
	link        Link
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

func newMux(link Link) *mux {
	m := &mux{ 
		link:        link, 
		flowsLocal:  make(map[uint64]*flow),
		flowsRemote: make(map[uint64]*flow),
		acceptChan:  make(chan *flow),
	}
	go m.loop()
	return m
}

func (m *mux) loop() {
	for {
		m.Lock()
		link := m.link
		m.Unlock()
		if link == nil {
			break
		}
		buf, addr, err := link.Read()
		if err != nil {
			break
		}
		msg, cargo, err := readMuxHeader(buf)
		if err != nil {
			continue
		}
		m.process(msg, cargo)
	}
	close(m.acceptChan)
	m.Lock()
	for _, f := range m.flowsLocal {
		close(f.ch)
	}
	for _, f := range m.flowsRemote {
		close(f.ch)
	}
	m.link = nil
	m.Unlock()
}

func (m *mux) process(msg *muxMsg, cargo []byte) {
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
		if f.getRemoteLabel() == nil {
			// If yes, then we just discovered the remote label. Save it.
			// Remark (*), above, ensures that the next 4 lines are executed atomically
			f.setRemoteLabel(msg.Source)
			m.Lock()
			m.flowsRemote[msg.Source] = f
			m.Unlock()
		} else if !f.getRemoteLabel().Equal(msg.Source) {
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

func (m *mux) accept(remoteLabel *Label, remoteAddr *Addr) *flow {
	if remoteLabel == nil {
		panic("remoteLabel == nil")
	}

	ch := make(chan muxHeader)
	localLabel := ChooseLabel()
	f := newFlow(remoteAddr, m, ch, localLabel, remoteLabel)

	m.Lock()
	m.flowsLocal[localLabel.Hash()] = f
	m.flowsRemote[remoteLabel.Hash()] = f
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
	f, closed := <-m.acceptChan
	if closed {
		return nil, os.EBADF
	}
	return f, nil
}

// findLocal() checks if there already exists a flow corresponding to the given local label
func (m *mux) findLocal(local *Label) *flow {
	if local == nil {
		return nil
	}
	m.lk.Lock()
	defer m.lk.Unlock()

	return m.flowsLocal[local.Hash()]
}

// findRemote() checks if there already exists a flow corresponding to the given remote label
func (m *mux) findRemote(remote *Label) *flow {
	if remote == nil {
		return nil
	}
	m.lk.Lock()
	defer m.lk.Unlock()

	return m.flowsRemote[remote.Hash()]
}

// addr@ is a textual representation of a link-level address, e.g.
//   0011`2233`4455`6677`8899`aabb`ccdd`eeff:453
func (m *mux) Dial(addr string) (c net.Conn, err os.Error) {
	a, err := ParseAddr(addr)
	if err != nil {
		return nil, err
	}
	return m.DialAddr(a)
}

func (m *mux) DialAddr(remoteAddr *Addr) (c net.Conn, err os.Error) {
	ch := make(chan muxHeader)
	localLabel := ChooseLabel()
	f = newFlow(remoteAddr, m, ch, localLabel, nil)

	m.Lock()
	m.flowsLocal[localLabel.Hash()] = f
	m.Unlock()

	return f, nil
}

// delFlow() removes the flow with the specified labels from the data structure
func (m *mux) delFlow(local *Label, remote *Label) {
	m.Lock()
	defer m.Unlock()

	if local != nil {
		m.flowsLocal[local.Hash()] = nil, false
	}
	if remote != nil {
		m.flowsRemote[remote.Hash()] = nil, false
	}
}

func (m *mux) write(msg *muxMsg, cargo []byte, remote *Addr) (n int, err os.Error) {
	m.Lock()
	link := m.link
	m.Unlock()
	if link == nil {
		return 0, os.EBADF
	}

	buf := make([]byte, msg.Len() + len(cargo))
	msg.Write(buf)
	copy(buf[msg.Len():], cargo)
	
	err, n = link.Write(buf, remote)
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
		close(f.ch)
	}
	for _, f := range m.flowsRemote {
		close(f.ch)
	}
	m.Unlock()
	if link == nil {
		return os.EBADF
	}
	return link.Close()
}

// Now() returns the current time in nanoseconds
func (m *mux) Now() int64 { return time.Now() }
