// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

/*
import (
	"net"
)


// Listener{} takes care of listening for incoming connections.
// It handles the LISTEN state.
type Listener struct {
	m     *Mux
	conns []*Conn	// List of active connections
			// TODO: Lookups in a short array should be fine for now. Hashing?
}

func Listen(m *Mux) net.Listener {
	l := &Listener{ m: m, }
	go l.loop()
	return l
}

// loop() reads and processes incoming packets
func (l *Listener) loop() {
	for {
		h,err := e.readPacket(l.phy, zeroFlowID)
		if err != nil {
			continue XX // no continue
		}
		?
	}
}

// readAndSwitch() reads an incoming packet and sends it over to
// its Conn{} destination if any, or returns it otherwise
func (l *listener) readAndSwitch() (h *Header, err os.Error) {
	h, err = read(l.phy)
	if err != nil {
		return 
	}
	?
}

// read() reads the next buffer of data from the link layer
// and tries to parse it into a valid Header{}
func read(r Reader) (*Header, os.Error) {

	// Read packet from physical layer
	buf, phyFlowID, err := r.Read()
	if err != nil {
		return nil, err
	}
	// Parse generic header
	h, err := ReadHeader(buf, zeroPhyFlowID.SourceAddr, zeroPhyFlowID.DestAddr, AnyProto, false)
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

func (l *Listener) Accept() (c Conn, err os.Error) {
	? //XX
	h,err := e.readPacket()
	if err != nil {
		return nil, err
	}

}

func (l *Listener) Close() os.Error {
	? //XX
	return l.phy.Close();
}

func (l *Listener) Addr() net.Addr { return DefaultAddr }

*/
