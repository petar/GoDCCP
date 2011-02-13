// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

func (h *GenericHeader) HasAckNo() bool { return getAckNoSubheaderSize(h.Type, h.X) > 0 }

func (h *GenericHeader) GetNDPCount() (int, os.Error) {
	?
}

func (h *GenericHeader) SetNDPCount(k int) {
	?
}

// NewResetHeader() creates a new Reset header
func NewResetHeader(ResetCode int, SourcePort, DestPort uint16, SeqNo, AckNo uint64) *GenericHeader {
	return &GenericHeader{
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
