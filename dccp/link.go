// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"io"
	"os"
	"strconv"
	"strings"
)

// Link{} is an abstract interface to a physical connection-less packet layer which sends and
// receives packets
type Link interface {
	Read() (buf []byte, addr *LinkAddr, err os.Error)	// Receive next packet of data
	Write(buf []byte, addr *LinkAddr) os.Error		// Send a packet of data
	io.Closer
}

// LinkAddr{} is the data type used to identify network nodes at the link layer
type LinkAddr struct {
	Label
	Port uint16
}

func (addr *LinkAddr) Network() string { return "godccp-link" }

func (addr *LinkAddr) Address() string {
	return addr.Label.String() + ":" + strconv.Itoa(int(addr.Port))
}

func (addr *LinkAddr) Parse(s string) (n int, err os.Error) {
	n, err = addr.Label.Parse(s)
	if err != nil {
		return 0, err
	}
	s = s[n:]
	if len(s) == 0 {
		return 0, os.NewError("link addr missing port")
	}
	if s[0] != ':' {
		return 0, os.NewError("link addr expecting ':'")
	}
	n += 1
	s = s[n:]
	q := strings.Index(s, " ")
	if q >= 0 {
		s = s[:q]
		n += q
	} else {
		n += len(s)
	}
	p, err := strconv.Atoui(s)
	if err != nil {
		return 0, err
	}
	addr.Port = uint16(p)
	return n, nil
}

func (addr *LinkAddr) Read(p []byte) (n int, err os.Error) {
	n, err = addr.Label.Read(p)
	if err != nil {
		return 0, err
	}
	p = p[n:]
	if len(p) < 2 {
		return 0, os.NewError("link addr missing port")
	}
	addr.Port = decode2ByteUint(p[0:2])
	return n+2, nil
}

func (addr *LinkAddr) Write(p []byte) (n int, err os.Error) {
	n, err = addr.Label.Write(p)
	if err != nil {
		return 0, err
	}
	p = p[n:]
	if len(p) < 2 {
		return 0, os.NewError("link addr can't fit port")
	}
	encode2ByteUint(addr.Port, p[0:2])
	return n+2, nil
}
