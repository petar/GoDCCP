// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
	"os"
	"time"
)

// flow{} is a BlockConn
type flow struct {
	addr    net.Addr
	m       *Mux
	ch      chan muxHeader
	largest int

	Mutex       // protects the variables below
	local       *Label
	remote      *Label
	lastRead    int64
	lastWrite   int64
	readTimeout int64

	rlk Mutex // synchronizes calls to Read()
}

func newFlow(addr net.Addr, m *Mux, ch chan muxHeader, largest int, local, remote *Label) *flow {
	now := m.Now()
	return &flow{
		addr:        addr,
		local:       local,
		remote:      remote,
		lastRead:    now,
		lastWrite:   now,
		readTimeout: 0,
		m:           m,
		ch:          ch,
		largest:     largest,
	}
}

// GetMTU returns the largest size of read/write block
func (f *flow) GetMTU() int { return f.largest }

// SetReadTimeout implements net.Conn.SetReadTimeout
func (f *flow) SetReadTimeout(nsec int64) os.Error {
	if nsec < 0 {
		return os.EINVAL
	}
	f.Lock()
	defer f.Unlock()
	f.readTimeout = nsec
	return nil
}

// LastRead() returns the timestamp of the last successful read operation
func (f *flow) LastReadTime() int64 {
	f.Lock()
	defer f.Unlock()
	return f.lastRead
}

// LastWrite() returns the timestamp of the last successful write operation
func (f *flow) LastWriteTime() int64 {
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

func (f *flow) WriteBlock(block []byte) os.Error {
	f.Lock()
	m := f.m
	f.Unlock()
	if m == nil {
		return os.EBADF
	}
	err := m.write(&muxMsg{f.getLocal(), f.getRemote()}, block, f.addr)
	if err != nil {
		f.Lock()
		f.lastWrite = m.Now()
		f.Unlock()
	}
	return err
}

func (f *flow) ReadBlock() (block []byte, err os.Error) {
	f.rlk.Lock()
	defer f.rlk.Unlock()

	f.Lock()
	ch := f.ch
	readTimeout := f.readTimeout
	f.Unlock()
	if ch == nil {
		return nil, os.EIO
	}

	var timer *time.Timer
	var tmoch <-chan int64
	if readTimeout > 0 {
		timer = time.NewTimer(readTimeout)
		defer timer.Stop()
		tmoch = timer.C
	}

	var header muxHeader
	var ok bool
	select {
	case header, ok = <-ch:
		if !ok {
			return nil, os.EIO
		}
	case <-tmoch:
		return nil, os.EAGAIN
	}

	f.Lock()
	if f.m != nil {
		f.lastRead = f.m.Now()
	}
	f.Unlock()

	return header.Cargo, nil
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
