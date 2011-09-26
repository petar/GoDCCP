// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package virtual

import (
	"os"
	"sync"
	"time"
	"github.com/petar/GoDCCP/dccp"
)

type Line struct {
	ha, hb headerHalfLine
}

func NewLine(sps uint32) (a, b dccp.HeaderConn, line *Line) {
	ab := make(chan *dccp.Header)
	ba := make(chan *dccp.Header)
	line = &Line{}
	line.ha.Init(ba, ab, sps)
	line.hb.Init(ab, ba, sps)
	return &line.ha, &line.hb, line
}

// headerHalfLine enforces rate-limiting on its write side
type headerHalfLine struct {
	read  <-chan *dccp.Header
	wlock sync.Mutex
	write chan<- *dccp.Header

	glock   sync.Mutex
	gap     int64 // Maximum time interval that can have at most one segment transmission
	gapCtr  int64 // UTC time in gap units
	gapFill int   // Number of segments transmitted during the gap in gapCtr
}

func (hhl *headerHalfLine) Init(r <-chan *dccp.Header, w chan<- *dccp.Header, sps uint32) {
	hhl.read = r
	hhl.write = w
	hhl.SetRate(sps)
}

func (hhl *headerHalfLine) GetMTU() int {
	return SegmentSize
}

func (hhl *headerHalfLine) ReadHeader() (h *dccp.Header, err os.Error) {
	h, ok := <-hhl.read
	if !ok {
		return nil, os.EOF
	}
	return h, nil
}

func (hhl *headerHalfLine) SetRate(segmentsPerSecond uint32) {
	hhl.glock.Lock()
	defer hhl.glock.Unlock()
	hhl.gap = 1e9 / int64(segmentsPerSecond)
	hhl.gapCtr = 0
	hhl.gapFill = 0
}

func (hhl *headerHalfLine) WriteHeader(h *dccp.Header) (err os.Error) {
	hhl.wlock.Lock()
	defer hhl.wlock.Unlock()
	if hhl.write == nil {
		return os.EBADF
	}
	
	if hhl.rateFilter() {
		hhl.write <- h
	}
	return nil
}

func (hhl *headerHalfLine) rateFilter() bool {
	hhl.glock.Lock()
	defer hhl.glock.Unlock()

	now := time.Nanoseconds()
	gctr := now / hhl.gap
	if gctr != hhl.gapCtr {
		hhl.gapCtr = gctr
		hhl.gapFill = 1
		return true
	} else if hhl.gapFill < 1 {
		hhl.gapFill++
		return true
	}
	return false
}

func (hhl *headerHalfLine) Close() os.Error { 
	hhl.wlock.Lock()
	defer hhl.wlock.Unlock()

	if hhl.write == nil {
		return os.EBADF
	}
	close(hhl.write)
	hhl.write = nil
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
