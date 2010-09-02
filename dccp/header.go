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
)

func isTypeReserved(type int) bool {
	return type >= 10 && type <= 15
}

// Reset codes
const (
	ResetUnspecified       = 0
	ResetClosed            = 1
	ResetAborted           = 2
	ResetNoConnection      = 3
	ResetPacketError       = 4
	ResetOptionError       = 5
	ResetMandatoryError    = 6
	ResetConnectionRefused = 7
	ResetBadServiceCode    = 8
	ResetTooBusy           = 9
	ResetBadInitCookie     = 10
	ResetAgressionPenalty  = 11
)

func isResetCodeReserved(code int) bool {
	return code >= 12 && code <= 127
}

func isResetCodeCCIDSpecific(code int) bool {
	return code >= 128 && code <= 255
}

var (
	ErrUnconstrained = os.NewError("unconstrained")
)

func unmarshalGenericHeader(buf []byte) (*genericHeader, os.Error) {
	?
}

func marshalGenericHeader(hdr *genericHeader) []byte {
	?
}

// If @err is nil, the value of @X can be determined from the @Type.
func calcX(Type int, AllowShortSeqNoFeature bool) (X int, err os.Error) {
	switch Type {
	case Request, Response:
		return 1, nil
	case Data, Ack, DataAck:
		if AllowShortSeqNoFeature {
			return 0, ErrUnconstrained
		}
		return 1, nil // X=1 means 48-bit (long) sequence numbers
	case CloseReq, Close:
		return 1, nil
	case Reset:
		return 1, nil
	case Sync, SyncAck:
		return 1, nil
	}
	panic("unreach")
}

// Any DCCP header has a subset of the following subheaders, in this order:
// + Generic header
// + Acknowledgement Number Subheader
// + Service Code, or Reset Code and Reset Data fields
// + Options and Padding
// + Application Data

// See RFC 4340, Page 21
func calcAckNoSubheaderSize(Type int, X int) int {
	if X != 0 && X != 1 {
		panic("logic")
	}
	if Type == Request || Type == Data {
		return 0
	}
	if X == 1 {
		return 8
	}
	return 4
}

func calcServiceCodeSize(Type int) int {
	switch Type {
	case Request, Response:
		return 4
	case Data, Ack, DataAck:
		return 0
	case CloseReq, Close:
		return 0
	case Reset:
		return 4
	case Sync, SyncAck:
		return 0
	}
	panic("unreach")
}

func mayHaveAppData(Type int) bool {
	switch Type {
	case Request, Response:
		return true
	case Data:
		return true
	case Ack:
		return true // may have App Data (essentially for padding) but must be ignored
	case DataAck:
		return true
	case CloseReq, Close:
		return true // may have App Data (essentially for padding) but must be ignored
	case Reset:
		return true // used for UTF-8 encoded error text
	case Sync, SyncAck:
		return true // may have App Data (essentially for padding) but must be ignored
	}
	panic("unreach")
}

