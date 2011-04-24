// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
	"os"
	"strconv"
	"strings"
)

// IP is a general-purpose address space (similar to IPv6)
type IP [IPLen]byte
const IPLen = 16

func (fip *IP) String() string {
	var w bytes.Buffer
	for i, b := range *fip {
		if i != 0 {
			w.WriteByte('`');
		}
		w.WriteString(btox(b))
	}
	return string(w.Bytes())
}

func (fip *IP) Read(p []byte) (n int, err os.Error) {
	if len(p) < IPLen {
		return 0, os.NewError("flow ip too short")
	}
	copy(fip[:], p[:IPLen])
	return IPLen, nil
}

func (fip *IP) Write(p []byte) (n int, err os.Error) {
	if len(p) < IPLen {
		return 0, os.NewError("flow ip can't fit")
	}
	copy(p, fip[:])
	return IPLen, nil
}

func (fip *IP) Parse(s string) (n int, err os.Error) {
	if len(s) < IPLen*3-1 {
		return 0, os.NewError("bad flow ip len")
	}
	for i := 0; i < IPLen; i++ {
		b, err := xtob(s[3*i:3*i+2])
		if err != nil {
			return 0, err
		}
		fip[i] = byte(b)
	}
	return IPLen*3-1, nil
}

const xalpha = "0123456789abcdef"
func btox(b byte) string {
	r := make([]byte, 2)
	r[1] = xalpha[b & 0xf]
	b >>= 4
	r[0] = xalpha[b & 0xf]
	return string(r)
}

func xtohb(a byte) (b byte, err os.Error) {
	if a >= '0' && a <= '9' {
		return a - '0', nil
	}
	if a >= 'a' && a <= 'f' {
		return 9 + a - 'a', nil
	}
	return 0, os.NewError("xtohb invalid char")
}

func xtob(s string) (b byte, err os.Error) {
	if len(s) != 2 {
		return 0, os.NewError("xtob len error")
	}
	b0, err := xtohb(s[1])
	if err != nil {
		return 0, err
	}
	b1, err := xtohb(s[0])
	if err != nil {
		return 0, err
	}
	return (b1 << 4) | b0, nil
}

// Addr{} combines a flow address and a port, which uniquely identifies a dial-to point
type Addr struct {
	IP   IP
	Port uint16
}

func (fa *Addr) Address() string {
	return fa.IP.String() + ":" + strconv.Itoa(int(fa.Port))
}

func (fa *Addr) Network() string { return "flow" }

func (fa *Addr) Parse(s string) (n int, err os.Error) {
	n, err = fa.IP.Parse(s)
	if err != nil {
		return 0, err
	}
	s = s[n:]
	if len(s) == 0 {
		return 0, os.NewError("flow addr missing port")
	}
	if s[0] != ':' {
		return 0, os.NewError("floaw addr expecting ':'")
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
	fa.Port = uint16(p)
	return n, nil
}

func (fa *Addr) Read(p []byte) (n int, err os.Error) {
	n, err = fa.IP.Read(p)
	if err != nil {
		return 0, err
	}
	p = p[n:]
	if len(p) < 2 {
		return 0, os.NewError("flow addr missing port")
	}
	fa.Port = decode2ByteUint(p[0:2])
	return n+2, nil
}

func (fa *Addr) Write(p []byte) (n int, err os.Error) {
	n, err = fa.IP.Write(p)
	if err != nil {
		return 0, err
	}
	p = p[n:]
	if len(p) < 2 {
		return 0, os.NewError("flow addr can't fit port")
	}
	encode2ByteUint(fa.Port, p[0:2])
	return n+2, nil
}

// ID{} contains identifiers of the local and remote logical addresses.
type ID struct {
	Source, Dest Addr
}

var ZeroID = ID{}

// Read reads the flow ID from the wire format
func (f *ID) Read(p []byte) (n int, err os.Error) {
	n0, err := f.Source.Read(p)
	if err != nil {
		return 0, err
	}
	n1, err := f.Dest.Read(p[n0:])
	if err != nil {
		return 0, err
	}
	return n0+n1, nil
}

// Write writes the flow ID in wire format
func (f *ID) Write(p []byte) (n int, err os.Error) {
	n0, err := f.Source.Write(p)
	if err != nil {
		return 0, err
	}
	n1, err := f.Dest.Read(p[n0:])
	if err != nil {
		return 0, err
	}
	return n0+n1, nil
}
