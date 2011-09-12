// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
	"strconv"
	"strings"
)

// Addr{} is a general-purpose data type intended for custom link layer addresses.
type Addr struct {
	*Label
	Port uint16
}

var ZeroAddr = &Addr{}

// Network() returns the name of the link address namespace, included to conform to net.Addr
func (addr *Addr) Network() string { return "godccp-addr" }

// String() returns the string represenation of the link address
func (addr *Addr) String() string {
	return addr.Label.String() + ":" + strconv.Itoa(int(addr.Port))
}

// Address() is identical to String(), included as a method so that Addr conforms to net.Addr
func (addr *Addr) Address() string { return addr.String() }

// ParseAddr() parses a link address from s@ in string format
func ParseAddr(s string) (addr *Addr, n int, err os.Error) {
	var label *Label
	label, n, err = ParseLabel(s)
	if err != nil {
		return nil, 0, err
	}
	s = s[n:]
	if len(s) == 0 {
		return nil, 0, os.NewError("link addr missing port")
	}
	if s[0] != ':' {
		return nil, 0, os.NewError("link addr expecting ':'")
	}
	n += 1
	s = s[1:]
	q := strings.Index(s, " ")
	if q >= 0 {
		s = s[:q]
		n += q
	} else {
		n += len(s)
	}
	p, err := strconv.Atoui(s)
	if err != nil {
		return nil, 0, err
	}
	return &Addr{label, uint16(p)}, n, nil
}

// Read() reads a link address from p@ in wire format
func ReadAddr(p []byte) (addr *Addr, n int, err os.Error) {
	var label *Label
	label, n, err = ReadLabel(p)
	if err != nil {
		return nil, 0, err
	}
	p = p[n:]
	if len(p) < 2 {
		return nil, 0, os.NewError("link addr missing port")
	}
	return &Addr{label, Decode2ByteUint(p[0:2])}, n + 2, nil
}

// Write() writes the link address to p@ in wire format
func (addr *Addr) Write(p []byte) (n int, err os.Error) {
	n, err = addr.Label.Write(p)
	if err != nil {
		return 0, err
	}
	p = p[n:]
	if len(p) < 2 {
		return 0, os.NewError("link addr can't fit port")
	}
	Encode2ByteUint(addr.Port, p[0:2])
	return n + 2, nil
}
