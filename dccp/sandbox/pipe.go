// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"fmt"
	"sync"
	"time"
	"github.com/petar/GoDCCP/dccp"
)

// Pipe is an in-process commincation channel, whose two ends implement dccp.HeaderConn.
// It supports rate limiting, latency emulation and receive buffer emulation (in order to
// capture slow readers).
type Pipe struct {
	amb *dccp.Amb
	ha, hb headerHalfPipe
}

// NewPipe creates a new pipe with a given runtime shared by both endpoints, and a root amb
func NewPipe(run *dccp.Runtime, amb *dccp.Amb, namea, nameb string) (a, b dccp.HeaderConn, line *Pipe) {
	ab := make(chan *pipeHeader, pipeBufferLen)
	ba := make(chan *pipeHeader, pipeBufferLen)
	line = &Pipe{}
	line.amb = amb
	line.ha.Init(run, line.amb.Refine(namea), ba, ab)
	line.hb.Init(run, line.amb.Refine(nameb), ab, ba)
	return &line.ha, &line.hb, line
}

const (
	DefaultRateInterval           = 1e9
	DefaultRatePacketsPerInterval = 100
)

const pipeBufferLen = 2

// headerHalfPipe implements HeaderConn. It enforces rate-limiting on its write side.
type headerHalfPipe struct {
	run                    *dccp.Runtime
	amb                    *dccp.Amb

	// read, writeLk and write pertain to the communication mechanism of the pipe
	read                   <-chan *pipeHeader
	writeLk                sync.Mutex
	write                  chan<- *pipeHeader

	// rateLk is used to lock on all rate* variables below as well as readDeadline
	rateLk                 sync.Mutex

	// Time is partitioned into equal units of rateInterval nanoseconds. Within each unit only
	// ratePacketsPerInterval packets are delivered; the rest are dropped
	rateInterval           int64      
	ratePacketsPerInterval uint32

	// rateIntervalCounter is the consequtive number of the current rateInterval interval
	rateIntervalCounter    int64

	// rateIntervalFill is the number of packets having been transmitted already during the
	// rateIntervalCounter-th time interval
	rateIntervalFill       uint32

	// readDeadline is the absolute time deadline for the reads on this side of the connection
	readDeadlineLk         sync.Mutex
	readDeadline           time.Time

	// writeLatency is the delay before non-dropped packets are delivered to the other side of the pipe
	latencyLk              sync.Mutex
	writeLatency           int64

	latencyQueue
}

type pipeHeader struct {
	Header      *dccp.Header
	DeliverTime int64
}

// Init resets a half pipe for initial use, using amb (without making a copy of it)
func (x *headerHalfPipe) Init(run *dccp.Runtime, amb *dccp.Amb, r <-chan *pipeHeader, w chan<- *pipeHeader) {
	x.run = run
	x.amb = amb
	x.read = r
	x.write = w
	x.SetWriteRate(DefaultRateInterval, DefaultRatePacketsPerInterval)
	x.readDeadline = time.Now().Add(-time.Second)
	x.writeLatency = 0
	x.latencyQueue.Init(run, amb)
}

// SetWriteLatency sets the write packet latency and it is given in nanoseconds
func (x *headerHalfPipe) SetWriteLatency(latency int64) {
	x.latencyLk.Lock()
	defer x.latencyLk.Unlock()
	x.writeLatency = latency
}

// SetWriteRate sets the transmission rate of this side of the pipe to ratePacketsPerInterval packets for each
// interval of rateInterval nanoseconds
func (x *headerHalfPipe) SetWriteRate(rateInterval int64, ratePacketsPerInterval uint32) {
	x.rateLk.Lock()
	defer x.rateLk.Unlock()
	x.rateInterval = rateInterval
	x.ratePacketsPerInterval = ratePacketsPerInterval
	x.rateIntervalCounter = 0
	x.rateIntervalFill = 0
}

// GetMTU implements dccp.HeaderConn.GetMTU
func (x *headerHalfPipe) GetMTU() int {
	return 1500
}

// ReadHeader implements dccp.HeaderConn.ReadHeader
func (x *headerHalfPipe) ReadHeader() (h *dccp.Header, err error) {
	x.readDeadlineLk.Lock()
	readDeadline := x.readDeadline
	x.readDeadlineLk.Unlock()

	sleepingChan := make(chan int64)

	for {
		XX?
		tmo := readDeadline.Sub(time.Now())
		var tmoch <-chan int64
		if tmo > 0 {
			tmoch = x.run.After(int64(tmo))
		} else {
			tmoch = sleepingChan
		}

		select {
		case ph, ok := <-x.read:
			if !ok {
				x.amb.E(dccp.EventWarn, "Read EOF", ph.Header)
				return nil, dccp.ErrEOF
			}
			x.amb.E(dccp.EventRead, fmt.Sprintf("SeqNo=%d", ph.Header.SeqNo), ph.Header)
			return ph.Header, nil
		case <-tmoch:
			return nil, dccp.ErrTimeout
		default:
			panic("un")
		}
	}
}

// WriteHeader implements dccp.HeaderConn.WriteHeader
func (x *headerHalfPipe) WriteHeader(h *dccp.Header) (err error) {
	x.writeLk.Lock()
	defer x.writeLk.Unlock()

	if x.write == nil {
		x.amb.E(dccp.EventDrop, fmt.Sprintf("ErrBad"), h)
		return dccp.ErrBad
	}

	if x.rateFilter() {
		if len(x.write) >= cap(x.write) {
			x.amb.E(dccp.EventDrop, "Slow reader", h)
		} else {
			x.amb.E(dccp.EventWrite, "", h)
			x.latencyLk.Lock()
			latency := x.writeLatency
			x.latencyLk.Unlock()
			now := x.run.Nanoseconds()
			x.write <- &pipeHeader{ Header: h, DeliverTime: now + latency }
		}
	} else {
		x.amb.E(dccp.EventDrop, "Fast writer", h)
	}
	return nil
}

// rateFilter returns true if another packet can be sent now without violating the rate
// limit set by SetWriteRate
func (x *headerHalfPipe) rateFilter() bool {
	x.rateLk.Lock()
	defer x.rateLk.Unlock()

	now := x.run.Nanoseconds()
	gctr := now / x.rateInterval
	if gctr != x.rateIntervalCounter {
		x.rateIntervalCounter = gctr
		x.rateIntervalFill = 1
		return true
	} else if x.rateIntervalFill < x.ratePacketsPerInterval {
		x.rateIntervalFill++
		return true
	}
	return false
}

// Close implements dccp.HeaderConn.Close
func (x *headerHalfPipe) Close() error {
	x.writeLk.Lock()
	defer x.writeLk.Unlock()

	if x.write == nil {
		x.amb.E(dccp.EventWarn, "Close EBADF")
		return dccp.ErrBad
	}
	close(x.write)
	x.write = nil

	x.amb.E(dccp.EventInfo, "Close")
	return nil
}

// LocalLabel implements dccp.HeaderConn.LocalLabel
func (x *headerHalfPipe) LocalLabel() dccp.Bytes {
	return &dccp.Label{}
}

// RemoteLabel implements dccp.HeaderConn.RemoteLabel
func (x *headerHalfPipe) RemoteLabel() dccp.Bytes {
	return &dccp.Label{}
}

// SetReadExpire implements dccp.HeaderConn.SetReadExpire
func (x *headerHalfPipe) SetReadExpire(nsec int64) error {
	x.readDeadlineLk.Lock()
	defer x.readDeadlineLk.Unlock()
	if nsec < 0 {
		panic("invalid timeout")
	}
	x.readDeadline = time.Now().Add(time.Duration(nsec))
	return nil
}
