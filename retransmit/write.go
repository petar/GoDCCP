// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package retransmit

import (
	"github.com/petar/GoDCCP/dccp"
)

// Header wire format
//
//     +------------+---------------+----------------+----------------+
//     | Type 1byte | Ack Subheader | Sync Subheader | Data Subheader |
//     +------------+---------------+----------------+----------------+
//
// Type-byte format
//
//     MSB           LSB
//     +-+-+-+-+-+-+-+-+
//     | | | | | |D|S|A|
//     +-+-+-+-+-+-+-+-+
//
// Ack Subheader wire format
//
//     +------------------+------------------+------------------+----------------+
//     | SyncAckNo 2bytes | AckMapLen 2bytes | DataAckNo 4bytes | ... AckMap ... |
//     +------------------+------------------+------------------+----------------+
//
// Sync Subheader wire format
//
//     +----------------+
//     | SyncNo 2 bytes |
//     +----------------+
//
// Data Subheader wire format
//
//     +---------------+----------------+--------------+
//     | DataNo 4bytes | DataLen 2bytes | ... Data ... |
//     +---------------+----------------+--------------+

// Write returns the wire format represenatation of h
func (h *header) Write() []byte {
}
