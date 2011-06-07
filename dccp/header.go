// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

// We use a subset of DCCP, defined as follows:
//	X=1
//      allowShortSeqNoFeature=0

// Any DCCP header has a subset of the following subheaders, in this order:
// (1a) Generic header
// (1b) Acknowledgement Number Subheader
// (1c) Code Subheader: Service Code, or Reset Code and Reset Data fields
// (2) Options and Padding
// (3) Application Data

type Header struct {
	SourcePort  uint16   // Sender port
	DestPort    uint16   // Receiver port
	CCVal       byte     // Used by the HC-Sender's CCID to transmit 4-bit values
	CsCov       byte     // Specifies the parts of packet covered by the checksum
	Type        byte     // Packet type: Data, Ack, Sync, etc.
	X           bool     // Extended seq numbers: generally always true (for us)
	SeqNo       int64    // 48-bit if X=1
	AckNo       int64    // 48-bit if X=1
	ServiceCode uint32   // ServiceCode: Applicaton level service (in Req,Resp pkts)
	ResetCode   byte     // ResetCode: Reason for reset (in Reset pkts)
	ResetData   []byte   // ResetData: Additional reset info (in Reset pkts)
	Options     []Option // Used for feature negotiation, padding, mandatory flags
	Data        []byte   // Application data (in Req, Resp, Data, DataAck pkts) 
	// Ignored (in Ack, Close, CloseReq, Sync, SyncAck pkts)
	// Error text (in Reset pkts)
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

// Constants for protoNo
const (
	AnyProto = iota
)

func isTypeUnderstood(Type byte) bool {
	return Type >= Request && Type <= SyncAck
}

func isTypeReserved(Type byte) bool {
	return Type >= 10 && Type <= 15
}

func isDataPacket(Type byte) bool {
	switch Type {
	case Request, Response, Data, DataAck:
		return true
	case Ack, Close, CloseReq, Reset, Sync, SyncAck:
		return false
	}
	panic("unreach")
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
	// Reserved 12 to 127
	// CCID-specific 128 to 255
)

func isResetCodeReserved(code int) bool {
	return code >= 12 && code <= 127
}

func isResetCodeCCIDSpecific(code int) bool {
	return code >= 128 && code <= 255
}

func areTypeAndXCompatible(Type byte, X bool, AllowShortSeqNoFeature bool) bool {
	switch Type {
	case Request, Response:
		return X
	case Data, Ack, DataAck:
		if AllowShortSeqNoFeature {
			return true
		}
		return X
	case CloseReq, Close:
		return X
	case Reset:
		return X
	case Sync, SyncAck:
		return X
	}
	panic("unreach")
}

// See RFC 4340, Page 21
func getAckNoSubheaderSize(Type byte, X bool) int {
	if Type == Request || Type == Data {
		return 0
	}
	if X {
		return 8
	}
	return 4
}

func getCodeSubheaderSize(Type byte) int {
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

// getFixedHeaderSize() returns the size of the fixed portion of
// the generic header in bytes, based on its @Type and @X. This
// includes (1a), (1b) and (1c).
func getFixedHeaderSize(Type byte, X bool) int {
	var r int
	switch X {
	case false:
		r = 12
	case true:
		r = 16
	}
	r += getAckNoSubheaderSize(Type, X)
	r += getCodeSubheaderSize(Type)
	return r
}

func mayHaveAppData(Type byte) bool {
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

// Checksum

const (
	CsCovAllData = iota
	CsCovNoData
	CsCov4
	CsCov8
	CsCov12
	// etc.
)

// getChecksumAppCoverage() computes how many bytes of the
// app data is covered by the checksum, not counting neccessary padding
func getChecksumAppCoverage(CsCov byte, dataLen int) (int, os.Error) {
	if CsCov > 15 {
		return 0, ErrCsCov
	}
	if CsCov == 0 {
		return dataLen, nil
	}
	cov := int(CsCov-1) << 2
	if cov > dataLen {
		return 0, ErrCsCov
	}
	return cov, nil
}
