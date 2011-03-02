// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp


// readPacket() reads the next buffer of data from the link layer
// and tries to parse it into a valid GenericHeader{}
func readPacket(r Reader, flowid FlowID) (*GenericHeader, os.Error) {

	// Read packet from physical layer
	buf, err := r.Read()
	if err != nil {
		return nil, err
	}
	// Parse generic header
	h, err := ReadGenericHeader(buf, flowid.SourceAddr, flowid.DestAddr, AnyProto, false)
	if err != nil {
		return nil, err
	}
	// Ensure extended SeqNo's 
	if !h.X {
		return nil, ErrUnsupported
	}
	return h, nil
}

func arrayEqual(a,b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (e *Endlogic) readPacket() (h *GenericHeader, err os.Error) {
	h, err = readPacket(e.phy, e.flowid)
	if err != nil {
		return nil, err
	}
	// Ensure source/dest address and port match with endpoint
	if !arrayEqual(h.SourceAddr, e.DestAddr) || !arrayEqual(h.DestAddr, e.SourceAddr) {
		return nil, ErrProto
	}
	if h.SourcePort != e.DestPort || h.DestPort != e.SourcePort {
		return nil, ErrProto
	}
	return h, nil
}

func (e *Endpoint) loop() {
	for {
		h,err := e.readPacket()
		if err != nil {
			continue XX // no continue
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
