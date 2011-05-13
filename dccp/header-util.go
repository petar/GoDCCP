// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

func (h *Header) HasAckNo() bool { return getAckNoSubheaderSize(h.Type, h.X) > 0 }

// NewResetHeader() creates a new Reset header
func NewResetHeader(resetCode uint32, sourcePort, destPort uint16) *Header {
	return &Header{
		SourcePort:  sourcePort,
		DestPort:    destPort,
		CsCov:       CsCovAllData,
		Type:        Reset,
		X:           true,
		ResetCode:   resetCode,
	}
}

// NewAckHeader() creates a new Ack header
func NewAckHeader(sourcePort, destPort uint16) *Header {
	return &Header{
		SourcePort:  sourcePort,
		DestPort:    destPort,
		Type:        Ack,
		X:           true,
	}
}

// NewSyncHeader() creates a new Sync header
func NewSyncHeader(sourcePort, destPort uint16) *Header {
	return &Header{
		SourcePort:  sourcePort,
		DestPort:    destPort,
		Type:        Sync,
		X:           true,
	}
}

// NewResponseHeader() creates a new Response header
func NewResponseHeader(serviceCode uint32, sourcePort, destPort uint16) *Header {
	return &Header{
		SourcePort:  sourcePort,
		DestPort:    destPort,
		Type:        Response,
		X:           true,
		ServiceCode: serviceCode,
	}
}
