// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp


// readPacket() reads the next buffer of data from the link layer
// and tries to parse it into a valid GenericHeader{}
func (e *Endlogic) readPacket() (*GenericHeader, os.Error) {

	// Read packet from physical layer
	buf, err := e.link.Recv()
	if err != nil {
		return nil, err
	}
	// Parse generic header
	h, err := ReadGenericHeader(buf, e.link.SourceIP(), e.link.DestIP(), AnyProto, false)
	if err != nil {
		return nil, err
	}
	// Ensure extended SeqNo's and source/dest port match with endpoint
	if !h.X {
		return nil, ErrUnsupported
	}
	if h.SourcePort != e.destPort || h.DestPort != e.SourcePort {
		return nil, ErrProto
	}
	return h, nil
}

func (e *Endpoint) loop() {
	for {
		if !e.link.Healthy() {
			break
		}
		h,err := e.readPacket()
		if err != nil {
			continue
		}
		?
	}
}

// If endpoint is in TIMEWAIT, it must perform a Reset sequence
func (e *Endpoint) reactInTIMEWAIT(h *GenericHeader) os.Error {
	if h.Type == Reset {
		return
	}
	var seqno uint64 = 0
	if h.HasAckNo() {
		seqno = h.AckNo+1
	}
	g := NewResetHeader(ResetNoConnection, e.flowID.SourcePort, e.flowID.DestPort, 
		seqno, h.SeqNo)
	hdr,data,err := g.Write(e.flowID.SourceAddr, e.flowID.DestAddr, AnyProto, false)
	if err != nil {
		return err
	}
	e.link.Send(append(hdr, data))
	return nil
}
