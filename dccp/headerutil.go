// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

func (h *Header) HasAckNo() bool { return getAckNoSubheaderSize(h.Type, h.X) > 0 }

func NewHeaderSkeleton(htype byte) *Header {
	return &Header{
		Type:       htype,
		X:          true,
	}
}

// NewResetHeader() creates a new Reset header
func NewResetHeader(resetCode byte) *Header {
	return &Header{
		Type:       Reset,
		X:          true,
		ResetCode:  resetCode,
	}
}

// NewCloseHeader() creates a new Close header
func NewCloseHeader() *Header {
	return &Header{
		Type:       Close,
		X:          true,
	}
}

// NewAckHeader() creates a new Ack header
func NewAckHeader() *Header {
	return &Header{
		Type:       Ack,
		X:          true,
	}
}

// NewDataHeader() creates a new Data header
func NewDataHeader(data []byte) *Header {
	return &Header{
		Type:       Data,
		X:          true,
		Data:       data,
	}
}

// NewDataAckHeader() creates a new DataAck header
func NewDataAckHeader(data []byte) *Header {
	return &Header{
		Type:       DataAck,
		X:          true,
		Data:       data,
	}
}

// NewSyncHeader() creates a new Sync header
func NewSyncHeader() *Header {
	return &Header{
		Type:       Sync,
		X:          true,
	}
}

// NewSyncAckHeader() creates a new Sync header
func NewSyncAckHeader() *Header {
	return &Header{
		Type:       SyncAck,
		X:          true,
	}
}

// NewRequestHeader() creates a new Request header
func NewRequestHeader(serviceCode uint32) *Header {
	return &Header{
		Type:        Request,
		X:           true,
		ServiceCode: serviceCode,
	}
}

// NewResponseHeader() creates a new Response header
func NewResponseHeader(serviceCode uint32) *Header {
	return &Header{
		Type:        Response,
		X:           true,
		ServiceCode: serviceCode,
	}
}
