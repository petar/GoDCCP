// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"sync"
	"github.com/petar/GoDCCP/dccp"
)

// NewHeaderPipe creates a new communication channel which has a HeaderConn interface on
// both sides.
func NewHeaderPipe() (ha, hb dccp.HeaderConn) {
	a := make(chan *dccp.Header)
	b := make(chan *dccp.Header)
	return &headerHalfPipe{read: a, write: b}, &headerHalfPipe{read: b, write: a}
}

type headerHalfPipe struct {
	read       <-chan *dccp.Header
	sync.Mutex // Lock for manipulating the write channel
	write      chan<- *dccp.Header
}

const SegmentSize = 1500

func (hhp *headerHalfPipe) GetMTU() int {
	return SegmentSize
}

func (hhp *headerHalfPipe) ReadHeader() (h *dccp.Header, err error) {
	h, ok := <-hhp.read
	if !ok {
		return nil, dccp.ErrEOF
	}
	return h, nil
}

func (hhp *headerHalfPipe) WriteHeader(h *dccp.Header) (err error) {
	hhp.Lock()
	defer hhp.Unlock()
	if hhp.write == nil {
		return dccp.ErrBad
	}
	hhp.write <- h
	return nil
}

func (hhp *headerHalfPipe) Close() error {
	hhp.Lock()
	defer hhp.Unlock()

	if hhp.write == nil {
		return dccp.ErrBad
	}
	close(hhp.write)
	hhp.write = nil
	return nil
}

func (hhp *headerHalfPipe) LocalLabel() dccp.Bytes {
	return &dccp.Label{}
}

func (hhp *headerHalfPipe) RemoteLabel() dccp.Bytes {
	return &dccp.Label{}
}

func (hhp *headerHalfPipe) SetReadExpire(nsec int64) error {
	panic("not implemented")
}
