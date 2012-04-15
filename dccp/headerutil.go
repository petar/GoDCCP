// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

func (h *Header) HasAckNo() bool { return getAckNoSubheaderSize(h.Type, h.X) > 0 }

// InitResetHeader() creates a new Reset header
func (h *Header) InitResetHeader(resetCode byte) {
	h.Type      = Reset
	h.X         = true
	h.ResetCode = resetCode
}

// InitCloseHeader() creates a new Close header
func (h *Header) InitCloseHeader() {
	h.Type = Close
	h.X    = true
}

// InitAckHeader() creates a new Ack header
func (h *Header) InitAckHeader() {
	h.Type = Ack
	h.X    = true
}

// InitDataHeader() creates a new Data header
func (h *Header) InitDataHeader(data []byte) {
	h.Type = Data
	h.X    = true
	h.Data = data
}

// InitDataAckHeader() creates a new DataAck header
func (h *Header) InitDataAckHeader(data []byte) {
	h.Type = DataAck
	h.X    = true
	h.Data = data
}

// InitSyncHeader() creates a new Sync header
func (h *Header) InitSyncHeader() {
	h.Type = Sync
	h.X    = true
}

// InitSyncAckHeader() creates a new Sync header
func (h *Header) InitSyncAckHeader() {
	h.Type = SyncAck
	h.X    = true
}

// InitRequestHeader() creates a new Request header
func (h *Header) InitRequestHeader(serviceCode uint32) {
	h.Type        = Request
	h.X           = true
	h.ServiceCode = serviceCode
}

// InitResponseHeader() creates a new Response header
func (h *Header) InitResponseHeader(serviceCode uint32) {
	h.Type        = Response
	h.X           = true
	h.ServiceCode = serviceCode
}
