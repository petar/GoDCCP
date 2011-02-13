// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"rand"
)

// XXX: Abstract the congestion control mechanism in a separate interface

type FlowID struct {
	SourceAddr, DestAddr	[]byte
	SourcePort, DestPort	uint16
}

// Endpoint logic for a single Half-connection
type Endpoint struct {
	link		Link
	flowID		FlowID

	RTT		uint64	// Round-trip time

	ISS		uint64	// Initial Sequence number Sent
	ISR		uint64	// Initial Sequence number Received
	GSS		uint64	// Greatest Sequence number Sent
				//	The greatest SeqNo of a packet sent by this endpoint
	GSR		uint64	// Greatest Sequence number Received (consequently, sent as AckNo back)
				//	The greatest SeqNo of a packet received by the other endpoint
	GAR		uint64	// Greatest Acknowledgement number Received

	OSR		uint64	// First OPEN Sequence number Received

	SWBF		uint64	// Sequence Window/B Feature
	SWAF		uint64	// Sequence Window/A Feature

	MPS		uint32	// Maximum Packet Size
					// The MPS is influenced by the
					// maximum packet size allowed by the current congestion control
					// mechanism (CCMPS), the maximum packet size supported by the path's
					// links (PMTU, the Path Maximum Transmission Unit) [RFC1191], and the
					// lengths of the IP and DCCP headers.

	State		int
	ServiceCode	uint32
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
	iss := pickInitialSeqNo()
	return &Endpoint{
		ISS: iss,
		GAR: iss,
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

// Connect() instructs the endpoint to initiate a connect request.
func (e *Endpoint) Connect() {
	?

	// Is CLOSED

	// Enter REQUEST

	// Send Request pkt

	// Wait for Response pkt

		// A client in the REQUEST state SHOULD use an exponential-backoff timer
		// to send new DCCP-Request packets if no response is received.  The
		// first retransmission should occur after approximately one second,
		// backing off to not less than one packet every 64 seconds; or the

		// A client MAY give up on its DCCP-Requests after some time (3 minutes,
		// for example).  When it does, it SHOULD send a DCCP-Reset packet to
		// the server with Reset Code 2, "Aborted", to clean up state in case
		// one or more of the Requests actually arrived.  A client in REQUEST
		// state has never received an initial sequence number from its peer, so
		// the DCCP-Reset's Acknowledgement Number MUST be set to zero.

	// When receive response, enter PARTOPEN

		// When the client receives a DCCP-Response from the server, it moves
		// from the REQUEST state to PARTOPEN and completes the three-way
		// handshake by sending a DCCP-Ack packet to the server.  The client
		// remains in PARTOPEN until it can be sure that the server has received
		// some packet the client sent from PARTOPEN (either the initial DCCP-
		// Ack or a later packet).  Clients in the PARTOPEN state that want to
		// send data MUST do so using DCCP-DataAck packets, not DCCP-Data
		// packets.  This is because DCCP-Data packets lack Acknowledgement
		// Numbers, so the server can't tell from a DCCP-Data packet whether the
		// client saw its DCCP-Response.

		// The single DCCP-Ack sent when entering the PARTOPEN state might, of
		// course, be dropped by the network.  The client SHOULD ensure that
		// some packet gets through eventually.  The preferred mechanism would
		// be a roughly 200-millisecond timer, set every time a packet is
		// transmitted in PARTOPEN.  If this timer goes off and the client is
		// still in PARTOPEN, the client generates another DCCP-Ack and backs
		// off the timer.  If the client remains in PARTOPEN for more than 4MSL
		// (8 minutes), it SHOULD reset the connection with Reset Code 2,
		// "Aborted".

	// Send an Ack or DataAck

	// Wait for first ack, then enter OPEN
		// The client leaves the PARTOPEN state for OPEN when it receives a
		// valid packet other than DCCP-Response, DCCP-Reset, or DCCP-Sync from
		// the server.
}

// Listen() handles the server-side listen logic for an endpoint
func (e *Endpoint) Listen() {
	?

	// Enter LISTEN

	// Wait for Request packet

	// Enter RESPOND

	// Wait for first ack, then enter OPEN


	// The server leaves the RESPOND state for OPEN when it receives a valid
	// DCCP-Ack from the client, completing the three-way handshake.  It MAY
	// also leave the RESPOND state for CLOSED after a timeout of not less
	// than 4MSL (8 minutes); when doing so, it SHOULD send a DCCP-Reset
	// with Reset Code 2, "Aborted", to clean up state at the client.
}

// While OPEN

   // DCCP A sends DCCP-Data and DCCP-DataAck packets to DCCP B due to
   // application events on host A.  These packets are congestion-
   // controlled by the CCID for the A-to-B half-connection.  In contrast,
   // DCCP-Ack packets sent by DCCP A are controlled by the CCID for the
   // B-to-A half-connection.  Generally, DCCP A will piggyback
   // acknowledgement information on DCCP-Data packets when acceptable,
   // creating DCCP-DataAck packets.  DCCP-Ack packets are used when there
   // is no data to send from DCCP A to DCCP B, or when the congestion
   // state of the A-to-B CCID will not allow data to be sent.

