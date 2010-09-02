// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

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

type genericHeader struct {
	SourcePort, DestPort uint16
	DataOffset           uint8
	CCVal, CsCov         uint8
	Checksum             uint16
	Res                  uint8
	Type                 uint8
	X                    bool
	Reserved             uint8
	SequenceNumber       uint64
}

// Packet types. Stored in the Type field of the generic header.
// Receivers MUST ignore any packets with reserved type.  That is,
// packets with reserved type MUST NOT be processed, and they MUST
// NOT be acknowledged as received.
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

func unmarshalGenericHeader(buf []byte) (*genericHeader, os.Error) {
	?
}

func marshalGenericHeader(hdr *genericHeader) []byte {
	?
}

// Any DCCP header has a subset of the following subheaders, in this order:
// + Generic header
// + Acknowledgement Number Subheader
// + Service Code
// + Options and Padding
// + Application Data

// See RFC 4340, Page 21
func calcAckNoSubheaderSize(Type int, X bool) int {
	if Type == Request || Type == Data {
		return 0
	}
	if X == 0 {
		return 4
	}
	if X == 1 {
		return 8
	}
	panic("unreach")
}

func calcServiceCodeSize(..) int {
}

func mayHaveAppData(Type int) bool {
	switch Type {
	case Request, Response:
		return true
	case ...
	}
	panic("unreach")
}

// RETURNS 0 or 1 if X is determined by Type, and -1 otherwise
func calcX(Type uint8) int {
	switch Type {
	case Request, Response:
		return 1
	case ..	
	}
	panic("unreach")
}
