// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

// DCCP is implemented after http://read.cs.ucla.edu/dccp/rfc4340.txt

package dccp

import (
	"os"
)

// Network byte order (most significant byte first)

const (
	Modulo = 1 << 48	// 2^48
)

// Circular-arithmetic
//
// "It may make sense to store DCCP
// sequence numbers in the most significant 48 bits of 64-bit integers
// and set the least significant 16 bits to zero, since this supports a
// common technique that implements circular comparison A < B by testing
// whether (A - B) < 0 using conventional two's-complement arithmetic."

// "Reserved bitfields in DCCP packet headers MUST be set to zero by
// senders and MUST be ignored by receivers, unless otherwise specified."

// Half-connections

// NOTATION: F/X = feature number and endpoint; in F/A: 
// 'feature location'=A, 'feature remote'=B

// Default round-trip time for use when no estimate is available
// This parameter should default to not less than 0.2 seconds.
const DefaultRoundtripTime = ?

// The maximum segment lifetime, or MSL, is the maximum length of time a
//  packet can survive in the network.  The DCCP MSL should equal that of
//  TCP, which is normally two minutes.
const MaximumSegmentLifetime = ?

// Connections progress through three phases: initiation, including a three-way
// handshake; data transfer; and termination.

// Packet types
const (
	Request  = 0
	Response = 1
	Data     = 2
	Ack      = 3
	DataAck  = 4
	CloseReq = 5
	Close    = 6
	Reset    = 7
	Sync     = 8
	SyncAck  = 9
	// Packet types 10-15 reserved
)

// DCCP sequence numbers increment by one per packet
type SequenceNumber uint64

// The nine possible states are as follows.  They are listed in
// increasing order, so that "state >= CLOSEREQ" means the same as
// "state = CLOSEREQ or state = CLOSING or state = TIMEWAIT".
const (
	CLOSED   = iota
	LISTEN   = _
	REQUEST  = _
	RESPOND  = _
	PARTOPEN = _
	OPEN     = _
	CLOSEREQ = _
	CLOSING  = _
	TIMEWAIT = _
)

// Congestion control mechanisms are denoted by one-byte 
// congestion control identifiers, or CCIDs.
// CCIDs 2 (TCP-like) and 3 (TFRC) are currently defined.

// There are four feature negotiation options in all: 
// Change L, Confirm L, Change R, and Confirm R.

//  The DCCP generic header takes different forms depending on the value
//  of X, the Extended Sequence Numbers bit.  If X is one, the Sequence
//  Number field is 48 bits long, and the generic header takes 16 bytes,
//  as follows.
//
//     0                   1                   2                   3
//     0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//    |          Source Port          |           Dest Port           |
//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//    |  Data Offset  | CCVal | CsCov |           Checksum            |
//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//    |     |       |X|               |                               .
//    | Res | Type  |=|   Reserved    |  Sequence Number (high bits)  .
//    |     |       |1|               |                               .
//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//    .  Sequence Number (low bits)                                   |
//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
//   If X is zero, only the low 24 bits of the Sequence Number are
//   transmitted, and the generic header is 12 bytes long.
//
//     0                   1                   2                   3
//     0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//    |          Source Port          |           Dest Port           |
//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//    |  Data Offset  | CCVal | CsCov |           Checksum            |
//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//    |     |       |X|                                               |
//    | Res | Type  |=|          Sequence Number (low bits)           |
//    |     |       |0|                                               |
//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type GenericHeader struct {
	SourcePort, DestPort uint16
	DataOffset           uint16
	CCVal, CsCov         uint8
	Checksum             uint16
	Res                  uint8
	Type                 uint8
	X                    bool
	Reserved             uint8
	SequenceNumber       uint64
}

// Up to:
// Kohler, et al.              Standards Track                    [Page 21]
