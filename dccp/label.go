// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
	"hash/crc64"
	"rand"
	"os"
	"strings"
)

// Label{} is an immutable object, representing a general-purpose address
// similar to IPv6 in that it consists of 16 opaque bytes.
type Label struct {
	data [LabelLen]byte
	h    uint64
}

const LabelLen = 16

var (
	LabelZero       = Label{}
	labelCRC64Table = crc64.MakeTable(crc64.ISO)
)

const labelFootprint = LabelLen

func (label *Label) Bytes() []byte {
	if label != nil {
		return label.data[:]
	}
	return LabelZero.data[:]
}

func (label *Label) hash() { 
	label.h = crc64.Checksum(label.data[:], labelCRC64Table) 
}

func isZero(bb []byte) bool {
	for _, b := range bb {
		if b != 0 {
			return false
		}
	}
	return true
}

// ChooseLabel() creates a new label by choosing its bytes randomly
func ChooseLabel() *Label {
	label := &Label{}
	for i := 0; i < LabelLen/2; i++ {
		q := rand.Int()
		label.data[2*i] = byte(q & 0xff)
		q >>= 8
		label.data[2*i+1] = byte(q & 0xff)
	}
	label.hash()
	return label
}

// Hash() returns the hash code of this label
func (label *Label) Hash() uint64 { 
	return label.h 
}

// Equal() performs a deep check for equality with q@
func (label *Label) Equal(q *Label) bool {
	for i := 0; i < LabelLen; i++ {
		if label.data[i] != q.data[i] {
			return false
		}
	}
	return true
}

// ReadLabel() reads and creates a new label from a wire format representation in p@
func ReadLabel(p []byte) (label *Label, n int, err os.Error) {
	if len(p) < LabelLen {
		return nil, 0, os.NewError("label too short")
	}
	label = &Label{}
	copy(label.data[:], p[:LabelLen])
	label.hash()
	if isZero(label.data[:]) {
		return nil, LabelLen, nil
	}
	return label, LabelLen, nil
}

// Write() writes the wire format representation of the label into p@
func (label *Label) Write(p []byte) (n int, err os.Error) {
	if label == nil {
		label = &LabelZero
	}
	if len(p) < LabelLen {
		return 0, os.NewError("label can't fit")
	}
	copy(p, label.data[:])
	return LabelLen, nil
}

// String() returns a string representation of the label
func (label *Label) String() string {
	if label == nil {
		label = &LabelZero
	}
	var w bytes.Buffer
	for i, b := range label.data {
		if i%2 == 0 && i > 0 {
			w.WriteByte('`')
		}
		w.WriteString(btox(b))
	}
	return string(w.Bytes())
}

func (label *Label) Address() string { 
	return label.String() 
}

// ParseLabel() parses and creates a new label from the string representation in s@
func ParseLabel(s string) (label *Label, n int, err os.Error) {
	l := LabelLen*2 + LabelLen/2 - 1
	if len(s) < l {
		return nil, 0, os.NewError("bad string label len")
	}
	s = strings.ToLower(s)
	label = &Label{}
	k := 0
	for i := 0; i < LabelLen; {
		if i%2 == 0 && i > 0 {
			if s[i] != '`' {
				return nil, 0, os.NewError("missing label dot")
			}
			i++
			continue
		}
		b, err := xtob(s[i : i+2])
		if err != nil {
			return nil, 0, err
		}
		i += 2
		label.data[k] = byte(b)
		k++
	}
	label.hash()
	if isZero(label.data[:]) {
		return nil, l, nil
	}
	return label, l, nil
}

const xalpha = "0123456789abcdef"

func btox(b byte) string {
	r := make([]byte, 2)
	r[1] = xalpha[b&0xf]
	b >>= 4
	r[0] = xalpha[b&0xf]
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
