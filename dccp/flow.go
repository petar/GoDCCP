// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
	"os"
	"sync"
)

// flow{} is a net.Conn{}
type flow struct {
	addr     net.Addr
	m        *Mux
	ch       chan muxHeader

	sync.Mutex // protects the variables below
	local     *Label
	remote    *Label
	lastRead  int64
	lastWrite int64

	rlk      sync.Mutex // synchronizes calls to Read()
}

func newFlow(addr net.Addr, m *Mux, ch chan muxHeader, local, remote *Label) *flow {
	now := m.Now()
	return &flow{
		addr:      addr,
		local:     local,
		remote:    remote,
		lastRead:  now,
		lastWrite: now,
		m:         m,
		ch:        ch,
	}
}

// LastRead() returns the timestamp of the last successful read operation
func (f *flow) LastRead() int64 {
	f.Lock()
	defer f.Unlock()
	return f.lastRead
}

// LastWrite() returns the timestamp of the last successful write operation
func (f *flow) LastWrite() int64 {
	f.Lock()
	defer f.Unlock()
	return f.lastWrite
}

func (f *flow) setRemote(remote *Label) {
	f.Lock()
	defer f.Unlock()
	if f.remote != nil {
		panic("setting remote label twice")
	}
	f.remote = remote
}

func (f *flow) getRemote() *Label {
	f.Lock()
	defer f.Unlock()
	return f.remote
}

func (f *flow) getLocal() *Label {
	f.Lock()
	defer f.Unlock()
	return f.local
}

func (f *flow) String() string {
	return f.getLocal().String() + "--" + f.getRemote().String()
}

func (f *flow) Write(cargo []byte) (n int, err os.Error) {
	f.Lock()
	m := f.m
	f.Unlock()
	if m == nil {
		return 0, os.EBADF
	}
	n, err =  m.write(&muxMsg{f.getLocal(), f.getRemote()}, cargo, f.addr)
	if err != nil {
		f.Lock()
		f.lastWrite = m.Now()
		f.Unlock()
	}
	return n, err
}

func (f *flow) Read(p []byte) (n int, err os.Error) {
	f.rlk.Lock()
	defer f.rlk.Unlock()

	f.Lock()
	ch := f.ch
	f.Unlock()
	if ch == nil {
		return 0, os.EIO
	}
	header, ok := <-ch
	if !ok {
		return 0, os.EIO
	}
	cargo := header.Cargo
	// TODO: This copy might be avoidable, since cargo@ is already an array dedicated to
	// this read, and allocated in Mux.loop()
	n = copy(p, cargo)
	if n != len(cargo) {
		panic("leftovers not desirable")
	}

	f.Lock()
	if f.m != nil {
		f.lastRead = f.m.Now()
	}
	f.Unlock()

	return n, nil
}

func (f *flow) foreclose() {
	f.Lock()
	defer f.Unlock()

	if f.ch != nil {
		close(f.ch)
		f.ch = nil
	}
}

func (f *flow) Close() os.Error {
	f.Lock()
	if f.ch != nil {
		close(f.ch)
		f.ch = nil
	}
	m := f.m
	f.m = nil
	f.Unlock()
	if m == nil {
		return os.EBADF
	}
	m.del(f.getLocal(), f.getRemote())
	return nil
}

func (f *flow) LocalAddr() net.Addr { return nil }

func (f *flow) RemoteAddr() net.Addr { return f.addr }

func (f *flow) SetTimeout(nsec int64) os.Error { panic("unimpl") }

func (f *flow) SetReadTimeout(nsec int64) os.Error { panic("unimpl") }

func (f *flow) SetWriteTimeout(nsec int64) os.Error { panic("unimpl") }
