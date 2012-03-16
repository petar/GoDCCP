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
	line.ha.Init(aName, run, line.logger, ba, ab, gap, packetsPerGap)
	line.hb.Init(bName, run, line.logger, ab, ba, gap, packetsPerGap)
	return &line.ha, &line.hb, line
}

// headerHalfLine implements HeaderConn. It enforces rate-limiting on its write side.
type headerHalfLine struct {
	name   string
	run    *dccp.Runtime
	logger *dccp.Logger

	read  <-chan *dccp.Header
	wlock sync.Mutex
	write chan<- *dccp.Header

	glock         sync.Mutex
	gap           int64      // Length of time interval for ...
	packetsPerGap uint32
	gapCounter    int64      // UTC time in gap units
	gapFill       uint32     // Number of segments transmitted during the gap in gapCounter
	deadline      time.Time  // Read deadline
}

func (hhl *headerHalfLine) Init(name string, run *dccp.Runtime, logger *dccp.Logger,
	r <-chan *dccp.Header, w chan<- *dccp.Header, gap int64, packetsPerGap uint32) {

	hhl.name = name
	hhl.run = run
	hhl.logger = logger
	hhl.read = r
	hhl.write = w
	hhl.SetRate(gap, packetsPerGap)
	hhl.deadline = time.Now().Add(-time.Second)
}

func (hhl *headerHalfLine) GetMTU() int {
	return SegmentSize
}

func (hhl *headerHalfLine) ReadHeader() (h *dccp.Header, err error) {
	hhl.glock.Lock()
	deadline := hhl.deadline
	hhl.glock.Unlock()
	tmo := deadline.Sub(time.Now())
	var tmoch <-chan int64
	if tmo > 0 {
		tmoch = hhl.run.After(int64(tmo))
	} else {
		tmoch = make(chan int64)
	}
	select {
	case h, ok := <-hhl.read:
		if !ok {
			hhl.logger.E(hhl.name, "Warn", "Read EOF", h)
			return nil, dccp.ErrEOF
		}
		hhl.logger.E(hhl.name, "Read", fmt.Sprintf("SeqNo=%d", h.SeqNo), h)
		return h, nil
	case <-tmoch:
		return nil, dccp.ErrTimeout
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
		hhl.logger.E(hhl.name, "Drop", fmt.Sprintf("SeqNo=%d EBADF", h.SeqNo), h)
		return dccp.ErrBad
	}

	if hhl.rateFilter() {
		if len(hhl.write) >= cap(hhl.write) {
			hhl.logger.E(hhl.name, "Drop", "Slow reader", h)
		} else {
			hhl.logger.E(hhl.name, "Write", fmt.Sprintf("SeqNo=%d", h.SeqNo), h)
			hhl.write <- h
		}
	} else {
		hhl.logger.E(hhl.name, "Drop", "Fast writer", h)
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
		hhl.logger.E(hhl.name, "Warn", "Close EBADF")
		return dccp.ErrBad
	}
	close(hhl.write)
	hhl.write = nil

	hhl.logger.E(hhl.name, "Event", "Close")
	return nil
}

func (hhl *headerHalfLine) LocalLabel() dccp.Bytes {
	return &dccp.Label{}
}

func (hhl *headerHalfLine) RemoteLabel() dccp.Bytes {
	return &dccp.Label{}
}

func (hhl *headerHalfLine) SetReadExpire(nsec int64) error {
	hhl.glock.Lock()
	defer hhl.glock.Unlock()
	if nsec < 0 {
		panic("invalid timeout")
	}
	hhl.deadline = time.Now().Add(time.Duration(nsec))
	return nil
}
