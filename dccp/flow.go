// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"io"
	"os"
	"sync"
)

// flow{} acts as a packet ReadWriteCloser{} for Conn.
type flow struct {
	sync.Mutex
	remote   *LinkAddr	// Link-layer address of the remote
	pair     *FlowPair	// Local and remote flow-layer keys
	m        *mux
	ch       chan muxHeader
	leftover []byte
}

func newFlow(remote *LinkAddr, m *mux, ch chan muxHeader, pair *FlowPair) *flow {
	return &flow{
		remote: remote,
		m:      mux,
		ch:     ch,
		pair:   pair,
	}
}

func (f *flow) Write(buf []byte) (n int, err os.Error) {
	f.Lock()
	m := f.m
	f.Unlock()
	if m == nil {
		return 0, os.EBADF
	}
	return m.write(buf, f.pair)
}

// Read() behaves unpredictably if calls occur concurrently.
func (f *flow) Read(p []byte) (n int, err os.Error) {
	f.Lock()
	if len(f.leftover) > 0 {
		n = copy(p, f.leftover)
		f.leftover = f.leftover[n:]
		f.Unlock()
		return n, nil
	}
	f.Unlock()

	header, closed := <-flow.ch
	if closed {
		return 0, os.EBADF
	}
	cargo := header.cargo
	n = copy(p, cargo)
	cargo = cargo[n:]
	if len(cargo) > 0 {
		f.Lock()
		f.leftover = cargo
		f.Unlock()
	}

	return n, nil
}

func (f *flow) Close() os.Error {
	close(f.ch)
	f.Lock()
	m = f.m
	f.m = nil
	f.Unlock()
	if m == nil {
		return os.EBADF
	}
	m.delFlow(f)
	return nil
}

func (f *flow) LocalAddr() net.Addr { return ZeroLinkAddr }

func (f *flow) RemoteAddr() net.Addr { return f.remote }

func (f *flow) SetTimeout(nsec int64) os.Error { panic("unimpl") }

func (f *flow) SetReadTimeout(nsec int64) os.Error { panic("unimpl") }

func (f *flow) SetWriteTimeout(nsec int64) os.Error { panic("unimpl") }
