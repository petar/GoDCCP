// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package retransmit

import (
	"os"
	"github.com/petar/GoDCCP/dccp"
)

func (h *header) getFootprintLen() int {
	a := 0
	if h.Ack {
		a = 2 + 2 + 4 + len(h.AckMap)
	}
	s := 0
	if h.Sync {
		s = 2
	}
	d := 0
	if h.Data {
		d = 4 + 2 + len(h.DataCargo)
	}
	return 1 + a + s + d
}

const (
	AckFlag = 1 << iota
	SyncFlag
	DataFlag
)

// Write returns the wire format represenatation of h
func (h *header) Write() (buf []byte, err os.Error) {
	buf = make([]byte, h.getFootprintLen())
	k := 0

	// Write type
	var t byte = 0
	if h.Ack {
		t |= AckFlag
	}
	if h.Sync {
		t |= SyncFlag
	}
	if h.Data {
		t |= DataFlag
	}
	buf[k] = t
	k++

	// Write Ack section
	if h.Ack {
		// Write SyncAckNo
		dccp.Encode2ByteUint(h.AckSyncNo, buf[k:k+2])
		k += 2

		// Write AckDataNo
		dccp.Encode4ByteUint(h.AckDataNo, buf[k:k+4])
		k += 4

		// Write AckMapLen
		if !dccp.FitsIn2Bytes(uint64(len(h.AckMap))) {
			return nil, dccp.ErrOverflow
		}
		dccp.Encode2ByteUint(uint16(len(h.AckMap)), buf[k:k+2])
		k += 2

		// Write AckMap
		copy(buf[k:k+len(h.AckMap)], h.AckMap)
		k += len(h.AckMap)
	}

	// Write Sync section
	if h.Sync {
		// Write SyncNo
		dccp.Encode2ByteUint(h.SyncNo, buf[k:k+2])
		k += 2
	}

	// Write Data section
	if h.Data {
		// Write DataNo
		dccp.Encode4ByteUint(h.DataNo, buf[k:k+4])
		k += 4

		// Write DataLen
		if !dccp.FitsIn2Bytes(uint64(len(h.DataCargo))) {
			return nil, dccp.ErrSize
		}
		dccp.Encode2ByteUint(uint16(len(h.DataCargo)), buf[k:k+2])
		k += 2

		// Write DataCargo
		copy(buf[k:k+len(h.DataCargo)], h.DataCargo)
		k += len(h.DataCargo)
	}

	return buf, nil
}
