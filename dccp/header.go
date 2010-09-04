// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

type GenericHeader struct {
	SourcePort, DestPort uint16
	DataOffset           uint8
	CCVal, CsCov         uint8
	Checksum             uint16
	Res                  uint8
	Type                 uint8
	X                    uint8
	Reserved             uint8
	SequenceNumber       uint64
}

var (
	ErrUnconstrained = os.NewError("unconstrained")
	ErrSize          = os.NewError("size")
)

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
// (1) Generic header
// (2) Acknowledgement Number Subheader
// (3) Code Subheader: Service Code, or Reset Code and Reset Data fields
// (4) Options and Padding
// (5) Application Data

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

func calcCodeSubheaderSize(Type int) int {
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

// calcFixedHeaderSize() returns the size of the fixed portion of
// the generic header in bytes, based on its @Type and @X. This
// includes (1), (2) and (3).
func calcFixedHeaderSize(Type int, X int) int {
	var r int
	switch X {
	case 0:
		r = 12
	case 1:
		r = 16
	default:
		panic("logic")
	}
	r += calcAckNoSubheaderSize(Type, X)
	r += calcCodeSubheaderSize(Type)
	return r
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

// Options

const (
	OptionPadding         = 0
	OptionMandatory       = 1
	OptionSlowReceiver    = 2
	OptionChangeL         = 32
	OptionConfirmL        = 33
	OptionChangeR         = 34
	OptionConfirmR        = 35
	OptionInitCookie      = 36
	OptionNDPCount        = 37
	OptionAckVectorNonce0 = 38
	OptionAckVectorNonce1 = 39
	OptionDataDropped     = 40
	OptionTimestamp       = 41
	OptionTimestampEcho   = 42
	OptionElapsedTime     = 43
	OptionDataChecksum    = 44
)

func isOptionReserved(optionType int) bool {
	return (optionType >= 3 && optionType <= 31) || 
		(optionType >= 45 && optionType <= 127)
}

func isOptionCCIDSpecific(optionType int) bool {
	return optionType >= 128 && optionType <= 255
}

func isOptionSingleByte(optionType int) bool {
	return optionType >= 0 && optionType <= 31
}

func isOptionValidForType(optionType, Type int) bool {
	if Type != Data {
		return true
	}
	switch optionType {
	case OptionPadding,
		OptionSlowReceiver,
		OptionNDPCount,
		OptionTimestamp,
		OptionTimestampEcho,
		OptionDataChecksum:
		return true
	default:
		return false
	}
	panic("unreach")
}

func (gh *GenericHeader) Write(hdr *GenericHeader) []byte {
	?
}
