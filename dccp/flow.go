// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
	"time"
)

// flow is an implementation of SegmentConn
type flow struct {
	addr net.Addr
	m    *Mux
	ch   chan muxHeader
	mtu  int

	Mutex        // protects the variables below
	local        *Label
	remote       *Label
	lastRead     time.Time
	lastWrite    time.Time
	readDeadline time.Time

	rlk Mutex // synchronizes calls to Read()
}

// addr is the Link-level address of the remote.
// local and remote are logical labels that are associated with each endpoint 
// of the connection. The remote label is not known until a packet is received
// from the other side.
func newFlow(addr net.Addr, m *Mux, ch chan muxHeader, mtu int, local, remote *Label) *flow {
	now := time.Now()
	return &flow{
		addr:         addr,
		local:        local,
		remote:       remote,
		lastRead:     now,
		lastWrite:    now,
		readDeadline: time.Now().Add(-time.Second),	// time in the past
		m:            m,
		ch:           ch,
		mtu:          mtu,
	}
}

// GetMTU returns the largest size of read/write block
func (f *flow) GetMTU() int { return f.mtu }

// SetReadExpire implements SegmentConn.SetReadExpire
func (f *flow) SetReadExpire(nsec int64) error {
	if nsec < 0 {
		return ErrInvalid
	}
	f.Lock()
	defer f.Unlock()
	f.readDeadline = time.Now().Add(time.Duration(nsec))
	return nil
}

// LastRead() returns the timestamp of the last successful read operation
func (f *flow) LastReadTime() time.Time {
	f.Lock()
	defer f.Unlock()
	return f.lastRead
}

// LastWrite() returns the timestamp of the last successful write operation
func (f *flow) LastWriteTime() time.Time {
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

// RemoteLabel implements SegmentConn.RemoteLabel
func (f *flow) RemoteLabel() Bytes { return f.getRemote() }

func (f *flow) getLocal() *Label {
	f.Lock()
	defer f.Unlock()
	return f.local
}

// LocalLabel implements SegmentConn.LocalLabel
func (f *flow) LocalLabel() Bytes { return f.getLocal() }

func (f *flow) String() string {
	return f.getLocal().String() + "--" + f.getRemote().String()
}

// Write implements SegmentConn.Write
func (f *flow) Write(block []byte) error {
	f.Lock()
	m := f.m
	f.Unlock()
	if m == nil {
		return ErrBad
	}
	err := m.write(&muxMsg{f.getLocal(), f.getRemote()}, block, f.addr)
	if err != nil {
		f.Lock()
		f.lastWrite = time.Now()
		f.Unlock()
	}
	return err
}

// Read implements SegmentConn.Read
func (f *flow) Read() (block []byte, err error) {
	f.rlk.Lock()
	defer f.rlk.Unlock()

	f.Lock()
	ch := f.ch
	readDeadline := f.readDeadline
	f.Unlock()
	readTimeout := readDeadline.Sub(time.Now())
	if ch == nil {
		return nil, ErrBad
	}

	var timer *time.Timer
	var tmoch <-chan time.Time
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
			return nil, ErrIO
		}
	case <-tmoch:
		return nil, ErrTimeout
	}

	f.Lock()
	f.lastRead = time.Now()
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

// Close implements SegmentConn.Close
func (f *flow) Close() error {
	f.Lock()
	if f.ch != nil {
		close(f.ch)
		f.ch = nil
	}
	m := f.m
	f.m = nil
	f.Unlock()
	if m == nil {
		return ErrBad
	}
	m.del(f.getLocal(), f.getRemote())
	return nil
}
