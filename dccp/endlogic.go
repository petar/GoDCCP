// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"rand"
)

// Endpoint logic for a single Half-connection
type Endpoint struct {
	RTT uint64	// Round-trip time

	ISS uint64	// Initial Sequence number Sent
	ISR uint64	// Initial Sequence number Received
	GSS uint64	// Greatest Sequence number Sent
			//	The greatest SeqNo of a packet sent by this endpoint
	GSR uint64	// Greatest Sequence number Received (consequently, sent as AckNo back)
			//	The greatest SeqNo of a packet received by the other endpoint
	GAR uint64	// Greatest Acknowledgement number Received

	SWBF uint64	// Sequence Window/B Feature
	SWAF uint64	// Sequence Window/A Feature

	State int
}

// CCID7 is the name we pick for our implementation of a congestion control 
// protocol over DCCP. CCID7 is described by:
//      (*) Only extended numbers are used
//      (*) No feature negitation mechanisms are implemented
//		(*) Send NDP Count feature is always ON
//		(*) Allow Short Sequence Numbers feature is always OFF

// The nine possible states are as follows.  They are listed in increasing order.
const (
	CLOSED   = iota
	LISTEN   = _
	REQUEST  = _
	RESPOND  = _
	PARTOPEN = _
	OPEN     = _
	CLOSEREQ = _
	CLOSING  = _
	TIMEWAIT = _
)

const (
	// Sequence Window Feature constants
	// XXX: Endpoints SHOULD send Change L(Sequence Window) options,
	// as necessary, as the connection progresses.
	// XXX: DCCP A MUST NOT persistently send more
	// than Sequence Window/A packets per RTT
	DefaultSWF_CCID7 = 100
	MinSWF = 32
	MaxSWF = 2^46-1
)

func NewEndpoint() *Endpoint {
	? // uninit'ed fields
	return &Endpoint{
		SWBF: DefaultSWF_CCID7,
		SWAF: DefaultSWF_CCID7,
	}
}

func pickInitialSeqNo() uint64 { return uint64(rand.Int63()) & 0xffffff }

func maxu64(x,y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}
// XXX: In theory, long lived connections may wrap around the AckNo/SeqNo space
// in which case maxu64() should not be used below. This will never happen however
// if we are using 48-bit numbers exclusively

// SWL and SWH
func (e *Endpoint) SeqNoWindowLowAndHigh() (SWL uint64, SWH uint64) {
	return maxu64(e.GSR + 1 - e.SWBF/4, e.ISR), e.GSR + (3*e.SWBF)/4
}

// AWL and AWH
func (e *Endpoint) AckNoWindowLowAndHigh() (AWL uint64, AWH uint64) {
	return maxu64(e.GSS + 1 - e.SWAF, e.ISS), e.GSS
}

func (e *Endpoint) IsSeqNoValid(h *GenericHeader) bool {
	swl, swh := e.SeqNoWindowLowHigh()
	switch h.Type {
	case Request, Response:
		if h.State == LISTEN || h.State == REQUEST {
			return true
		}
		return swl <= h.SeqNo && h.SeqNo <= swh
	case Data, Ack, DataAck:
		return swl <= h.SeqNo && h.SeqNo <= swh
	case CloseReq, Close, Reset:
		return h.GSR <= h.SeqNo && h.SeqNo <= swh
	case Sync, SyncAck:
		if e.IsActive() {
			return swl <= h.SeqNo && h.SeqNo <= swh
		}
		return swl <= h.SeqNo
	}
	panic("unreach")
}

func (e *Endpoint) IsAckNoValid(h *GenericHeader) bool {
	awl, awh := e.AckNoWindowLowHigh()
	switch h.Type {
	case Request, Data:
		return true
	case Response, Ack, DataAck, Sync, SyncAck:
		return awl <= h.AckNo && h.AckNo <= awh
	case CloseReq, Close, Reset:
		return e.GAR <= h.AckNo && h.AckNo <= awh
	}
	panic("unreach")
}

func (e *Endpoint) IsAckSeqNoValid(h *GenericHeader) bool {
	return e.IsAckNoValud(h) && e.isSeqNoValid(h)

// A connection is considered active if it has received valid packets 
// from the other endpoint within the last three round-trip times
// AckNo and SeqNo validity checks are more stringent for active connections.
func (e *Endpoint) IsActive() bool {
	return TimeNow() - e.???? < 3 * e.RTT
}

// TimeNow() returns the current time at this endpoint
func (e *Endpoint) TimeNow() uint64 {
	??
}

// OnReceive() is called by the physical layer to hand off a received
// packet to the endpoint for processing.
// TODO:
//	Optimization against bursts:
//		...  In particular, an endpoint MAY preserve sequence-
//		invalid packets for up to 2 round-trip times.  If, within that time,
//		the relevant sequence windows change so that the packets become
//		sequence-valid, the endpoint MAY process them again. ...
func (e *Endpoint) OnReceive(h *GenericHeader) {
	?

	ackSeqOk := e.IsAckSeqNoValid(h)

	if !ackSeqOk {
		// Any sequence-invalid DCCP-Sync or DCCP-SyncAck packet MUST be ignored.
		if h.Type == Sync || h.Type == SyncAck
			return // ignore
		}
		
		if h.Type == Reset {
		      // A sequence-invalid DCCP-Reset packet MUST elicit a DCCP-Sync
		      // packet in response (subject to a possible rate limit).  This
		      // response packet MUST use a new Sequence Number, and thus will
		      // increase GSS; GSR will not change, however, since the received
		      // packet was sequence-invalid.  The response packet's
		      // Acknowledgement Number MUST equal GSR.
		      ?
		}

		// Any other sequence-invalid packet MUST elicit a similar DCCP-Sync
		// packet, except that the response packet's Acknowledgement Number
		// MUST equal the sequence-invalid packet's Sequence Number.
		?
	}

	// Now process the sequence valid packets
	switch h.Type {
	case Sync:
		h.onSync(h)
		?
	}
	panic("unreach")
}

func (e *Endpoint) onSync(h *GenericHeader) {
	// On receiving a sequence-valid DCCP-Sync packet, the peer endpoint
	// (say, DCCP B) MUST update its GSR variable and reply with a DCCP-
	// SyncAck packet.  The DCCP-SyncAck packet's Acknowledgement Number
	// will equal the DCCP-Sync's Sequence Number, which is not necessarily
	// GSR.  Upon receiving this DCCP-SyncAck, which will be sequence-valid
	// since it acknowledges the DCCP-Sync, DCCP A will update its GSR
	// variable, and the endpoints will be back in sync.  As an exception,
	// if the peer endpoint is in the REQUEST state, it MUST respond with a
	// DCCP-Reset instead of a DCCP-SyncAck.  This serves to clean up DCCP
	// A's half-open connection.
	?

	// To protect against denial-of-service attacks, DCCP implementations
	// SHOULD impose a rate limit on DCCP-Syncs sent in response to
	// sequence-invalid packets, such as not more than eight DCCP-Syncs per
	// second.
}


// Send() let's the endpoint know that the application layer above wants to
// send the given buffer of application data in a packet to the other endpoint.
func (e *Endpoint) Send(buf []byte) {
   // DCCP's sequence numbers increment by one on every packet, including
   // non-data packets (packets that don't carry application data).
   ?

   // If a DCCP endpoint's Send NDP Count feature is one (see below), then
   // that endpoint MUST send an NDP Count option on every packet whose
   // immediate predecessor was a non-data packet.
   ?
}

func (e *Endpoint) NDPCount() uint64 {
   // The value stored in NDP Count equals the number of consecutive non-
   // data packets in the run immediately previous to the current packet.
   // Packets with no NDP Count option are considered to have NDP Count
   // zero.
   ?
}
// With NDP Count, the receiver can reliably tell only whether a burst
// of loss contained at least one data packet.  For example, the
// receiver cannot always tell whether a burst of loss contained a non-
// data packet.
