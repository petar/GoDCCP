// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

func (h *Header) HasAckNo() bool { return getAckNoSubheaderSize(h.Type, h.X) > 0 }

func (h *Header) GetNDPCount() (int, os.Error) {
	panic("¿i?")
}

func (h *Header) SetNDPCount(k int) {
	panic("¿i?")
}

// NewResetHeader() creates a new Reset header
func NewResetHeader(ResetCode uint32, SourcePort, DestPort uint16, SeqNo, AckNo uint64) *Header {
	return &Header{
		SourcePort:  SourcePort,
		DestPort:    DestPort,
		CsCov:       CsCovAllData,
		Type:        Reset,
		X:           true,
		SeqNo:       SeqNo,
		AckNo:       AckNo,
		ServiceCode: ResetCode,
	}
}
