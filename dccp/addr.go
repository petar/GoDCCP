// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
	"rand"
	"os"
	"strconv"
	"strings"
)

// Label is a general-purpose address space (similar to IPv6)
type Label [LabelLen]byte
const LabelLen = 16

func (label *Label) Choose() {
	for i := 0; i < LabelLen/2; i++ {
		q := rand.Int()
		label[2*i] = byte(q & 0xff)
		label[2*i+1] = byte(q & 0xff00)
	}
}

func (label *Label) Equal(q *Label) bool { 
	for i := 0; i < LabelLen; i++ {
		if label[i] != q[i] {
			return false
		}
	}
	return true
}

func (label *Label) Read(p []byte) (n int, err os.Error) {
	if len(p) < LabelLen {
		return 0, os.NewError("label too short")
	}
	copy(label[:], p[:LabelLen])
	return LabelLen, nil
}

func (label *Label) Write(p []byte) (n int, err os.Error) {
	if len(p) < LabelLen {
		return 0, os.NewError("label can't fit")
	}
	copy(p, label[:])
	return LabelLen, nil
}

func (label *Label) String() string {
	var w bytes.Buffer
	for i, b := range *label {
		if i % 2 == 0 && i > 0 {
			w.WriteByte('`');
		}
		w.WriteString(btox(b))
	}
	return string(w.Bytes())
}

func (label *Label) Address() string { return label.String() }

func (label *Label) Parse(s string) (n int, err os.Error) {
	l := LabelLen*2 + LabelLen/2 - 1 
	if len(s) < l {
		return 0, os.NewError("bad label len")
	}
	k := 0
	for i := 0; i < LabelLen; {
		if i % 2 == 0 && i > 0 {
			if s[i] != '`' {
				return 0, os.NewError("missing label dot")
			}
			i++
			continue
		}
		b, err := xtob(s[i:i+2])
		if err != nil {
			return 0, err
		}
		i += 2
		label[k] = byte(b)
		k++
	}
	return LabelLen*3-1, nil
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

// Each end-point of a flow assigns its own FlowKey{} to the flow
type FlowKey struct {
	Label
}

func (fkey *FlowKey) Network() string { return "godccp-flow" }

// FlowPair{} contains identifiers of the local and remote logical addresses.
type FlowPair struct {
	Source, Dest FlowKey
}

var ZeroFlowPair = FlowPair{}

// Read reads the FlowPair from the wire format
func (fpair *FlowPair) Read(p []byte) (n int, err os.Error) {
	n0, err := fpair.Source.Read(p)
	if err != nil {
		return 0, err
	}
	n1, err := fpair.Dest.Read(p[n0:])
	if err != nil {
		return 0, err
	}
	return n0+n1, nil
}

// Write writes the FlowPair in wire format
func (fpair *FlowPair) Write(p []byte) (n int, err os.Error) {
	n0, err := fpair.Source.Write(p)
	if err != nil {
		return 0, err
	}
	n1, err := fpair.Dest.Read(p[n0:])
	if err != nil {
		return 0, err
	}
	return n0+n1, nil
}

// Addr{} combines a link-layer address and a user-level port
type Addr struct {
	Addr  LinkAddr
	Port  uint16
}

func (addr *Addr) Address() string {
	return addr.Addr.String() + ":" + strconv.Itoa(int(addr.Port))
}

func (addr *Addr) Network() string { return "godccp" }

func (addr *Addr) Parse(s string) (n int, err os.Error) {
	n, err = addr.Addr.Parse(s)
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
	n, err = addr.Addr.Read(p)
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
	n, err = addr.Addr.Write(p)
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
