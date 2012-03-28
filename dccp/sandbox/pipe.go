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
// It supports rate limiting and latency emulation.
type Pipe struct {
	logger *dccp.Logger
	ha, hb headerHalfPipe
}

func NewPipe(run *dccp.Runtime, logger *dccp.Logger, 
	aName, bName string, rateInterval int64, ratePacketsPerInterval uint32) (a, b dccp.HeaderConn, line *Pipe) {

	ab := make(chan *pipeHeader, pipeBufferLen)
	ba := make(chan *pipeHeader, pipeBufferLen)
	line = &Pipe{}
	line.logger = logger
	line.ha.Init(aName, run, line.logger, ba, ab, rateInterval, ratePacketsPerInterval)
	line.hb.Init(bName, run, line.logger, ab, ba, rateInterval, ratePacketsPerInterval)
	return &line.ha, &line.hb, line
}

const (
	DefaultRateInterval           = 1e9
	DefaultRatePacketsPerInterval = 100
)

const pipeBufferLen = 2

// headerHalfPipe implements HeaderConn. It enforces rate-limiting on its write side.
type headerHalfPipe struct {
	name   string
	run    *dccp.Runtime
	logger *dccp.Logger

	// read, writeLk and write pertain to the communication mechanism of the pipe
	read  <-chan *pipeHeader
	writeLk sync.Mutex
	write chan<- *pipeHeader

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

	// latency is the delay before non-dropped packets are delivered to the other side of the pipe
	latencyLk sync.Mutex
	latency   int64

	// latencyReadQueue is a buffer of packets internally received at this end of the pipe,
	// annotated with a lower bound on the time before they can be delivered to ReadHeader
	latencyReadQueueLk sync.Mutex
	latencyReadQueue   []*pipeHeader
}

type pipeHeader struct {
	Header      *dccp.Header
	DeliverTime int64
}

func (hhl *headerHalfPipe) Init(name string, run *dccp.Runtime, logger *dccp.Logger,
	r <-chan *dccp.Header, w chan<- *dccp.Header) {

	hhl.name = name
	hhl.run = run
	hhl.logger = logger
	hhl.read = r
	hhl.write = w
	hhl.SetWriteRate(DefaultRateInterval, DefaultRatePacketsPerInterval)
	hhl.readDeadline = time.Now().Add(-time.Second)
	hhl.latency = 0
	hhl.latencyReadQueue = make([]*pipeHeader, 0)
}

// SetWriteLatency sets the write packet latency and it is given in nanoseconds
func (hhl *headerHalfPipe) SetWriteLatency(latency int64) {
	hhl.latencyLk.Lock()
	defer hhl.latencyLk.Unlock()
	hhl.latency = latency
}

// SetWriteRate sets the transmission rate of this side of the pipe to ratePacketsPerInterval packets for each
// interval of rateInterval nanoseconds
func (hhl *headerHalfPipe) SetWriteRate(rateInterval int64, ratePacketsPerInterval uint32) {
	hhl.rateLk.Lock()
	defer hhl.rateLk.Unlock()
	hhl.rateInterval = rateInterval
	hhl.ratePacketsPerInterval = ratePacketsPerInterval
	hhl.rateIntervalCounter = 0
	hhl.rateIntervalFill = 0
}

// GetMTU implements dccp.HeaderConn.GetMTU
func (hhl *headerHalfPipe) GetMTU() int {
	return 1500
}

// GetMTU implements dccp.HeaderConn.ReadHeader
func (hhl *headerHalfPipe) ReadHeader() (h *dccp.Header, err error) {
	hhl.readDeadlineLk.Lock()
	readDeadline := hhl.readDeadline
	hhl.readDeadlineLk.Unlock()
	tmo := readDeadline.Sub(time.Now())
	var tmoch <-chan int64
	if tmo > 0 {
		tmoch = hhl.run.After(int64(tmo))
	} else {
		tmoch = make(chan int64)
	}
	select {
	case ph, ok := <-hhl.read:
		if !ok {
			hhl.logger.E(hhl.name, "Warn", "Read EOF", ph.Header)
			return nil, dccp.ErrEOF
		}
		hhl.logger.E(hhl.name, "Read", fmt.Sprintf("SeqNo=%d", h.SeqNo), ph.Header)
		return ph.Header, nil
	case <-tmoch:
		return nil, dccp.ErrTimeout
	}
	panic("un")
}

func (hhl *headerHalfPipe) WriteHeader(h *dccp.Header) (err error) {
	hhl.writeLk.Lock()
	defer hhl.writeLk.Unlock()

	if hhl.write == nil {
		hhl.logger.E(hhl.name, "Drop", fmt.Sprintf("EBADF"), h)
		return dccp.ErrBad
	}

	if hhl.rateFilter() {
		if len(hhl.write) >= cap(hhl.write) {
			hhl.logger.E(hhl.name, "Drop", "Slow reader", h)
		} else {
			hhl.logger.E(hhl.name, "Write", "", h)
			hhl.write <- &pipeHeader{ Header: h }
		}
	} else {
		hhl.logger.E(hhl.name, "Drop", "Fast writer", h)
	}
	return nil
}

func (hhl *headerHalfPipe) rateFilter() bool {
	hhl.rateLk.Lock()
	defer hhl.rateLk.Unlock()

	now := hhl.run.Nanoseconds()
	gctr := now / hhl.rateInterval
	if gctr != hhl.rateIntervalCounter {
		hhl.rateIntervalCounter = gctr
		hhl.rateIntervalFill = 1
		return true
	} else if hhl.rateIntervalFill < hhl.ratePacketsPerInterval {
		hhl.rateIntervalFill++
		return true
	}
	return false
}

func (hhl *headerHalfPipe) Close() error {
	hhl.writeLk.Lock()
	defer hhl.writeLk.Unlock()

	if hhl.write == nil {
		hhl.logger.E(hhl.name, "Warn", "Close EBADF")
		return dccp.ErrBad
	}
	close(hhl.write)
	hhl.write = nil

	hhl.logger.E(hhl.name, "Event", "Close")
	return nil
}

func (hhl *headerHalfPipe) LocalLabel() dccp.Bytes {
	return &dccp.Label{}
}

func (hhl *headerHalfPipe) RemoteLabel() dccp.Bytes {
	return &dccp.Label{}
}

// SetReadExpire implements dccp.HeaderConn.SetReadExpire
func (hhl *headerHalfPipe) SetReadExpire(nsec int64) error {
	hhl.readDeadlineLk.Lock()
	defer hhl.readDeadlineLk.Unlock()
	if nsec < 0 {
		panic("invalid timeout")
	}
	hhl.readDeadline = time.Now().Add(time.Duration(nsec))
	return nil
}
