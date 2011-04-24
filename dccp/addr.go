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

func (ip *IP) Equal(q *IP) bool { 
	for i := 0; i < IPLen; i++ {
		if ip[i] != q[i] {
			return false
		}
	}
	return true
}

func (ip *IP) Read(p []byte) (n int, err os.Error) {
	if len(p) < IPLen {
		return 0, os.NewError("ip too short")
	}
	copy(ip[:], p[:IPLen])
	return IPLen, nil
}

func (ip *IP) Write(p []byte) (n int, err os.Error) {
	if len(p) < IPLen {
		return 0, os.NewError("ip can't fit")
	}
	copy(p, ip[:])
	return IPLen, nil
}

func (ip *IP) String() string {
	var w bytes.Buffer
	for i, b := range *ip {
		if i % 2 == 0 && i > 0 {
			w.WriteByte('`');
		}
		w.WriteString(btox(b))
	}
	return string(w.Bytes())
}

func (ip *IP) Parse(s string) (n int, err os.Error) {
	l := IPLen*2 + IPLen/2 - 1 
	if len(s) < l {
		return 0, os.NewError("bad ip len")
	}
	k := 0
	for i := 0; i < IPLen; {
		if i % 2 == 0 && i > 0 {
			if s[i] != '`' {
				return 0, os.NewError("missing ip dot")
			}
			i++
			continue
		}
		b, err := xtob(s[i:i+2])
		if err != nil {
			return 0, err
		}
		i += 2
		ip[k] = byte(b)
		k++
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

// Addr{} combines an IP and a port, which uniquely identifies a dial-to point
type Addr struct {
	IP   IP
	Port uint16
}

func (addr *Addr) Address() string {
	return addr.IP.String() + ":" + strconv.Itoa(int(addr.Port))
}

func (addr *Addr) Network() string { return "godccp" }

func (addr *Addr) Parse(s string) (n int, err os.Error) {
	n, err = addr.IP.Parse(s)
	if err != nil {
		return 0, err
	}
	s = s[n:]
	if len(s) == 0 {
		return 0, os.NewError("addr missing port")
	}
	if s[0] != ':' {
		return 0, os.NewError("addr expecting ':'")
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

func (addr *Addr) Read(p []byte) (n int, err os.Error) {
	n, err = addr.IP.Read(p)
	if err != nil {
		return 0, err
	}
	p = p[n:]
	if len(p) < 2 {
		return 0, os.NewError("addr missing port")
	}
	addr.Port = decode2ByteUint(p[0:2])
	return n+2, nil
}

func (addr *Addr) Write(p []byte) (n int, err os.Error) {
	n, err = addr.IP.Write(p)
	if err != nil {
		return 0, err
	}
	p = p[n:]
	if len(p) < 2 {
		return 0, os.NewError("addr can't fit port")
	}
	encode2ByteUint(addr.Port, p[0:2])
	return n+2, nil
}

// ID{} contains identifiers of the local and remote logical addresses.
type ID struct {
	Source, Dest Addr
}

var ZeroID = ID{}

// Read reads the ID from the wire format
func (id *ID) Read(p []byte) (n int, err os.Error) {
	n0, err := id.Source.Read(p)
	if err != nil {
		return 0, err
	}
	n1, err := id.Dest.Read(p[n0:])
	if err != nil {
		return 0, err
	}
	return n0+n1, nil
}

// Write writes the ID in wire format
func (id *ID) Write(p []byte) (n int, err os.Error) {
	n0, err := id.Source.Write(p)
	if err != nil {
		return 0, err
	}
	n1, err := id.Dest.Read(p[n0:])
	if err != nil {
		return 0, err
	}
	return n0+n1, nil
}
