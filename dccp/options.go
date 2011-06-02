// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

type Option struct {
	Type      byte
	Data      []byte
	Mandatory bool
}

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
	// Reserved 45 to 127
	// CCID-specific 128 to 255
)

func isOptionReserved(optionType byte) bool {
	return (optionType >= 3 && optionType <= 31) ||
		(optionType >= 45 && optionType <= 127)
}

func isOptionCCIDSpecific(optionType byte) bool {
	return optionType >= 128 && optionType <= 255
}

func isOptionSingleByte(optionType byte) bool {
	return optionType >= 0 && optionType <= 31
}

func isOptionValidForType(optionType, Type byte) bool {
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
