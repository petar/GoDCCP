// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package retransmit

import (
	"os"
	"github.com/petar/GoDCCP/dccp"
)

// readHeader parses a header from its wire representation
func readHeader(buf []byte) (h *header, err os.Error) {

	h = &header{}
	k := 0

	// Read type
	if len(buf) < 1 {
		return nil, dccp.ErrSize
	}
	t := buf[k]
	k++
	if t & AckFlag != 0 {
		h.Ack = true
	}
	if t & SyncFlag != 0 {
		h.Sync = true
	}
	if t & DataFlag != 0 {
		h.Data = true
	}

	// Read Ack section
	if h.Ack {
		if len(buf) - k < 2 + 4 + 2 {
			return nil, dccp.ErrSize
		}

		// Read AckSyncNo
		h.AckSyncNo = dccp.Decode2ByteUint(buf[k:k+2])
		k += 2

		// Read AckDataNo
		h.AckDataNo = dccp.Decode4ByteUint(buf[k:k+4])
		k += 4

		// Read AckMapLen
		ackMapLen := dccp.Decode2ByteUint(buf[k:k+2])
		k += 2

		if len(buf) - k < int(ackMapLen) {
			return nil, dccp.ErrSize
		}

		// Read AckMap
		h.AckMap = buf[k:k+int(ackMapLen)]
		k += int(ackMapLen)
	}

	// Read Sync section
	if h.Sync {
		if len(buf) - k < 2 {
			return nil, dccp.ErrSize
		}

		// Read SyncNo
		h.SyncNo = dccp.Decode2ByteUint(buf[k:k+2])
		k += 2
	}

	// Read Data section
	if h.Data {
		if len(buf) - k < 4 + 2 {
			return nil, dccp.ErrSize
		}

		// Read DataNo
		h.DataNo = dccp.Decode4ByteUint(buf[k:k+4])
		k += 4

		// Read DataLen
		dataLen := dccp.Decode2ByteUint(buf[k:k+2])
		k += 2

		if len(buf) - k < int(dataLen) {
			return nil, dccp.ErrSize
		}

		// Read DataCargo
		h.DataCargo = buf[k:k+int(dataLen)]
		k += int(dataLen)
	}

	return h, nil
}
