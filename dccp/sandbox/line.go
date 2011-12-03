// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"io"
	"os"
	"sync"
	"github.com/petar/GoDCCP/dccp"
)

type Line struct {
	logger *dccp.Logger
	ha, hb headerHalfLine
}

const LineBufferLen = 2

func NewLine(run *dccp.Runtime, logger *dccp.Logger, 
	aName, bName string, gap int64, packetsPerGap uint32) (a, b dccp.HeaderConn, line *Line) {

	ab := make(chan *dccp.Header, LineBufferLen)
	ba := make(chan *dccp.Header, LineBufferLen)
	line = &Line{}
	line.logger = logger
	line.ha.Init(aName, run, line.logger, ba, ab, gap, packetsPerGap, 1e9)
	line.hb.Init(bName, run, line.logger, ab, ba, gap, packetsPerGap, 1e9)
	return &line.ha, &line.hb, line
}

// headerHalfLine enforces rate-limiting on its write side
type headerHalfLine struct {
	name   string
	run    *dccp.Runtime
	logger *dccp.Logger

	read  <-chan *dccp.Header
	wlock sync.Mutex
	write chan<- *dccp.Header

	glock         sync.Mutex
	gap           int64  // Length of time interval for ...
	packetsPerGap uint32
	gapCounter    int64  // UTC time in gap units
	gapFill       uint32 // Number of segments transmitted during the gap in gapCounter
	timeout       int64  // Read timeout in nanoseconds
}

func (hhl *headerHalfLine) Init(name string, run *dccp.Runtime, logger *dccp.Logger,
	r <-chan *dccp.Header, w chan<- *dccp.Header, gap int64, packetsPerGap uint32,
	tmo int64) {

	hhl.name = name
	hhl.run = run
	hhl.logger = logger
	hhl.read = r
	hhl.write = w
	hhl.SetRate(gap, packetsPerGap)
	hhl.timeout = tmo
}

func (hhl *headerHalfLine) GetMTU() int {
	return SegmentSize
}

func (hhl *headerHalfLine) ReadHeader() (h *dccp.Header, err error) {
	hhl.glock.Lock()
	tmo := hhl.timeout
	hhl.glock.Unlock()
	var tmoch <-chan int64
	if tmo > 0 {
		tmoch = hhl.run.After(tmo)
	} else {
		tmoch = make(chan int64)
	}
	select {
	case h, ok := <-hhl.read:
		if !ok {
			hhl.logger.Emit(hhl.name, "Warn", h, "Read EOF")
			return nil, io.EOF
		}
		hhl.logger.Emit(hhl.name, "Read", h, "SeqNo=%d", h.SeqNo)
		return h, nil
	case <-tmoch:
		return nil, os.EAGAIN
	}
	panic("un")
}

func (hhl *headerHalfLine) SetRate(gap int64, packetsPerGap uint32) {
	hhl.glock.Lock()
	defer hhl.glock.Unlock()
	hhl.gap = gap
	hhl.packetsPerGap = packetsPerGap
	hhl.gapCounter = 0
	hhl.gapFill = 0
}

func (hhl *headerHalfLine) WriteHeader(h *dccp.Header) (err error) {
	hhl.wlock.Lock()
	defer hhl.wlock.Unlock()

	if hhl.write == nil {
		hhl.logger.Emit(hhl.name, "Drop", h, "SeqNo=%d EBADF", h.SeqNo)
		return os.EBADF
	}

	if hhl.rateFilter() {
		if len(hhl.write) >= cap(hhl.write) {
			hhl.logger.Emit(hhl.name, "Drop", h, "Slow reader")
		} else {
			hhl.logger.Emit(hhl.name, "Write", h, "SeqNo=%d", h.SeqNo)
			hhl.write <- h
		}
	} else {
		hhl.logger.Emit(hhl.name, "Drop", h, "Fast writer")
	}
	return nil
}

func (hhl *headerHalfLine) rateFilter() bool {
	hhl.glock.Lock()
	defer hhl.glock.Unlock()

	now := hhl.run.Nanoseconds()
	gctr := now / hhl.gap
	if gctr != hhl.gapCounter {
		hhl.gapCounter = gctr
		hhl.gapFill = 1
		return true
	} else if hhl.gapFill < hhl.packetsPerGap {
		hhl.gapFill++
		return true
	}
	return false
}

func (hhl *headerHalfLine) Close() error {
	hhl.wlock.Lock()
	defer hhl.wlock.Unlock()

	if hhl.write == nil {
		hhl.logger.Emit(hhl.name, "Warn", nil, "Close EBADF")
		return os.EBADF
	}
	close(hhl.write)
	hhl.write = nil

	hhl.logger.Emit(hhl.name, "Event", nil, "Close")
	return nil
}

func (hhl *headerHalfLine) LocalLabel() dccp.Bytes {
	return &dccp.Label{}
}

func (hhl *headerHalfLine) RemoteLabel() dccp.Bytes {
	return &dccp.Label{}
}

func (hhl *headerHalfLine) SetReadTimeout(nsec int64) error {
	hhl.glock.Lock()
	defer hhl.glock.Unlock()
	if nsec < 0 {
		panic("invalid timeout")
	}
	hhl.timeout = nsec
	return nil
}
