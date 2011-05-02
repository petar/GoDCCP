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
	m        *mux
	ch       chan muxHeader

	sync.Mutex // protects local and remote
	local    *Label
	remote   *Label

	rlk      sync.Mutex // protects leftover and calls to Read()
	leftover []byte
}

func newFlow(addr net.Addr, m *mux, ch chan muxHeader, local, remote *Label) *flow {
	return &flow{
		addr:   addr,
		local:  local,
		remote: remote,
		m:      m,
		ch:     ch,
	}
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

func (f *flow) Write(cargo []byte) (n int, err os.Error) {
	f.Lock()
	m := f.m
	f.Unlock()
	if m == nil {
		return 0, os.EBADF
	}
	return m.write(&muxMsg{f.getLocal(), f.getRemote()}, cargo, f.addr)
}

func (f *flow) Read(p []byte) (n int, err os.Error) {
	f.rlk.Lock()
	defer f.rlk.Unlock()

	if len(f.leftover) > 0 {
		n = copy(p, f.leftover)
		f.leftover = f.leftover[n:]
		return n, nil
	}

	header, ok := <-f.ch
	if !ok {
		return 0, os.EIO
	}
	cargo := header.Cargo
	n = copy(p, cargo)
	cargo = cargo[n:]
	if len(cargo) > 0 {
		f.leftover = cargo
	}

	return n, nil
}

func (f *flow) Close() os.Error {
	close(f.ch)
	f.Lock()
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
