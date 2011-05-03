// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
	"os"
)

// ChanLink{} treats one side of a channel as an incoming packet link
type ChanLink struct {
	in, out  chan []byte
}

func NewChanPipe() (p,q *ChanLink) {
	c0 := make(chan []byte)
	c1 := make(chan []byte)
	return &ChanLink{ c0, c1 }, &ChanLink{ c1, c0 }
}

func (l *ChanLink) FragmentLen() int { return 1500 }

func (l *ChanLink) ReadFrom(buf []byte) (n int, addr net.Addr, err os.Error) {
	p, ok := <- l.in
	if !ok {
		return 0, nil, os.EIO
	}
	n = copy(buf, p)
	if n != len(p) {
		panic("insufficient buf len")
	}
	return len(p), nil, nil
}

func (l *ChanLink) WriteTo(buf []byte, addr net.Addr) (n int, err os.Error) {
	p := make([]byte, len(buf))
	copy(p, buf)
	l.out <- p
	return len(buf), nil
}

func (l *ChanLink) Close() os.Error {
	close(l.in)
	close(l.out)
	return nil
}
