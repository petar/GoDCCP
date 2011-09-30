// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"os"
	"sync"
	"time"
	"github.com/petar/GoDCCP/dccp"
)

// TODO: Specify rate by (interval, maximum # packets per interval)

type Line struct {
	dccp.CLog
	ha, hb headerHalfLine
}

func NewLine(name string, na, nb string, gap int64, packetsPerGap uint32) (a, b dccp.HeaderConn, line *Line) {
	ab := make(chan *dccp.Header)
	ba := make(chan *dccp.Header)
	line = &Line{}
	line.CLog.Init(name)
	line.ha.Init(na, line.CLog, ba, ab, gap, packetsPerGap)
	line.hb.Init(nb, line.CLog, ab, ba, gap, packetsPerGap)
	return &line.ha, &line.hb, line
}

// headerHalfLine enforces rate-limiting on its write side
type headerHalfLine struct {
	name string
	dccp.CLog

	read  <-chan *dccp.Header
	wlock sync.Mutex
	write chan<- *dccp.Header

	glock   sync.Mutex
	gap           int64  // Length of time interval for ...
	packetsPerGap uint32
	gapCounter    int64  // UTC time in gap units
	gapFill       uint32 // Number of segments transmitted during the gap in gapCounter
}

func (hhl *headerHalfLine) Init(name string, clog dccp.CLog, 
	r <-chan *dccp.Header, w chan<- *dccp.Header, gap int64, packetsPerGap uint32) {

	hhl.name = name
	hhl.CLog = clog
	hhl.read = r
	hhl.write = w
	hhl.SetRate(gap, packetsPerGap)
}

func (hhl *headerHalfLine) GetMTU() int {
	return SegmentSize
}

func (hhl *headerHalfLine) ReadHeader() (h *dccp.Header, err os.Error) {
	h, ok := <-hhl.read
	if !ok {
		hhl.CLog.Logf(hhl.name, "Warn", "Read EOF")
		return nil, os.EOF
	}
	hhl.CLog.Logf(hhl.name, "Read", "SeqNo=%d", h.SeqNo)
	return h, nil
}

func (hhl *headerHalfLine) SetRate(gap int64, packetsPerGap uint32) {
	hhl.glock.Lock()
	defer hhl.glock.Unlock()
	hhl.gap = gap
	hhl.packetsPerGap = packetsPerGap
	hhl.gapCounter = 0
	hhl.gapFill = 0
}

func (hhl *headerHalfLine) WriteHeader(h *dccp.Header) (err os.Error) {
	hhl.wlock.Lock()
	defer hhl.wlock.Unlock()

	if hhl.write == nil {
		hhl.CLog.Logf(hhl.name, "Drop", "SeqNo=%d EBADF", h.SeqNo)
		return os.EBADF
	}
	
	if hhl.rateFilter() {
		hhl.write <- h
		hhl.CLog.Logf(hhl.name, "Write", "SeqNo=%d", h.SeqNo)
	} else {
		hhl.CLog.Logf(hhl.name, "Drop", "SeqNo=%d", h.SeqNo)
	}
	return nil
}

func (hhl *headerHalfLine) rateFilter() bool {
	hhl.glock.Lock()
	defer hhl.glock.Unlock()

	now := time.Nanoseconds()
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

func (hhl *headerHalfLine) Close() os.Error { 
	hhl.wlock.Lock()
	defer hhl.wlock.Unlock()

	if hhl.write == nil {
		hhl.CLog.Logf(hhl.name, "Warn", "Close EBADF")
		return os.EBADF
	}
	close(hhl.write)
	hhl.write = nil

	hhl.CLog.Logf(hhl.name, "Event", "Close")
	return nil
}

func (hhl *headerHalfLine) LocalLabel() dccp.Bytes { 
	return &dccp.Label{}
}

func (hhl *headerHalfLine) RemoteLabel() dccp.Bytes { 
	return &dccp.Label{}
}

func (hhl *headerHalfLine) SetReadTimeout(nsec int64) os.Error { 
	return nil
}