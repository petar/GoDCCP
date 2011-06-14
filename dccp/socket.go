// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
	"fmt"
	"rand"
)

// socket is a data structure, maintaining the DCCP socket variables.
// socket's methods are not re-entrant
type socket struct {
	ISS int64 // Initial Sequence number Sent
	ISR int64 // Initial Sequence number Received

	OSR int64 // First OPEN Sequence number Received

	// Here and elsewhere, "greatest" is measured in circular sequence space (modulo 2^48)
	GSS int64 // Greatest Sequence number Sent

	GSR int64 // Greatest valid Sequence number Received (consequently, sent as AckNo back)
	GAR int64 // Greatest valid Acknowledgement number Received on a non-Sync; initialized to S.ISS

	CCIDA byte // CCID in use for the A-to-B half-connection (aka HC-Sender CCID), Section 10
	CCIDB byte // CCID in use for the B-to-A half-connection (aka HC-Receiver CCID), Section 10

	// Sequence Window/A Feature, see Section 7.5.1
	// Controls the width of the Acknowledgement Number validity window used by the local DCCP
	// endpoint (DCCP A), and the width of Sequence Number validity window used by the remote
	// DCCP endpoint (DCCP B)
	//
	// A proper Sequence Window/A value must reflect the number of packets DCCP A expects to be
	// in flight.  Only DCCP A can anticipate this number.
	//
	// XXX: One good guideline is for each endpoint to set Sequence Window to about five times
	// the maximum number of packets it expects to send in a round- trip time.  Endpoints SHOULD
	// send Change L(Sequence Window) options, as necessary, as the connection progresses.
	//
	// XXX: Also, an endpoint MUST NOT persistently send more than its Sequence Window number of
	// packets per round-trip time; that is, DCCP A MUST NOT persistently send more than
	// Sequence Window/A packets per RTT.
	SWAF int64 	
	
	// Sequence Window/B Feature, see Section 7.5.1
	// Controls the width of the Sequence Number validity window used by the local DCCP
	// endpoint (DCCP A), and the width of Acknowledgement Number validity window used by the remote
	// DCCP endpoint (DCCP B)
	SWBF int64 

	State       int
	Server      bool   // True if the endpoint is a server, false if it is a client
	ServiceCode uint32 // The service code of this connection

	PMTU  int32 // Path Maximum Transmission Unit
	CCMPS int32 // Congestion Control Maximum Packet Size

	RTT int64 // Round Trip Time in nanoseconds
}

func (s *socket) String() string {
	var w bytes.Buffer
	fmt.Fprintf(&w, "State=%s:%s %s, ISS=%d, ISR=%d, OSR=%d, GSS=%d, GSR=%d, GAR=%d, SWAF=%d, SWBF=%d, RTT=%d",
		StateString(s.State), ServerString(s.Server), ServiceCodeString(s.ServiceCode),
		s.ISS, s.ISR, s.OSR, s.GSS, s.GSR, s.GAR, s.SWAF, s.SWBF, s.RTT)
	return string(w.Bytes())
}

const (
	SEQWIN_INIT            = 100      // Initial value for SWAF and SWBF, Section 7.5.2
	SEQWIN_FIXED           = 700      // Large enough constant for SWAF/SWBF until the feature is implemented
	SEQWIN_MAX             = 2^46 - 1 // Maximum acceptable SWAF and SWBF value
	RTT_DEFAULT            = 2e8      // 0.2 sec, default Round-Trip Time when no measurement is available
	MSL                    = 2 * 60e9 // 2 mins in nanoseconds, Maximum Segment Lifetime, Section 3.4
	CLOSING_BACKOFF_FREQ   = 64e9     // Backoff frequency of CLOSING timer, 64 seconds, Section 8.3
	CLOSING_BACKOFF_MAX    = MSL      // Maximum amount of time in CLOSING timer
	MAX_OPTIONS_SIZE       = 128
)

// The nine possible states of a DCCP socket.  Listed in increasing order:
const (
	CLOSED = iota
	LISTEN
	REQUEST
	RESPOND
	PARTOPEN
	OPEN
	CLOSEREQ
	CLOSING
	TIMEWAIT
)

func StateString(state int) string {
	switch state {
	case CLOSED:
		return "CLOSED"
	case LISTEN:
		return "LISTEN"
	case REQUEST:
		return "REQUEST"
	case RESPOND:
		return "RESPOND"
	case PARTOPEN:
		return "PARTOPEN"
	case OPEN:
		return "OPEN"
	case CLOSEREQ:
		return "CLOSEREQ"
	case CLOSING:
		return "CLOSING"
	case TIMEWAIT:
		return "TIMEWAIT"
	}
	panic("unreach")
}

func ServerString(isServer bool) string {
	if isServer {
		return "Server"
	}
	return "Client"
}

func (s *socket) SetCCIDA(v byte) { s.CCIDA = v }
func (s *socket) SetCCIDB(v byte) { s.CCIDB = v }

func (s *socket) GetMPS() int32 { return min32(s.CCMPS, s.PMTU) }

func (s *socket) GetPMTU() int32  { return s.PMTU }
func (s *socket) SetPMTU(v int32) { s.PMTU = v }

func (s *socket) GetCCMPS() int32  { return s.CCMPS }
func (s *socket) SetCCMPS(v int32) { s.CCMPS = v }

func (s *socket) GetRTT() int64  { return s.RTT }
func (s *socket) SetRTT(v int64) { s.RTT = v }

func (s *socket) SetServer(v bool) { s.Server = v }
func (s *socket) IsServer() bool   { return s.Server }

func (s *socket) GetState() int  { return s.State }
func (s *socket) SetState(v int) { s.State = v }

func (s *socket) SetServiceCode(v uint32) { s.ServiceCode = v }
func (s *socket) GetServiceCode() uint32  { return s.ServiceCode }

// ChooseISS chooses a safe Initial Sequence Number
func (s *socket) ChooseISS() int64 {
	iss := rand.Int63n(0xffffff-1) + 1
	s.ISS = iss
	return iss
}
func (s *socket) GetISS() int64 { return s.ISS }

func (s *socket) SetISR(v int64) { s.ISR = v }

func (s *socket) GetOSR() int64  { return s.OSR }
func (s *socket) SetOSR(v int64) { s.OSR = v }

func (s *socket) GetGSS() int64  { return s.GSS }
func (s *socket) SetGSS(v int64) { s.GSS = v }

func (s *socket) GetGSR() int64     { return s.GSR }
func (s *socket) SetGSR(v int64)    { s.GSR = v }
func (s *socket) UpdateGSR(v int64) { s.GSR = max64(s.GSR, v) }

func (s *socket) GetGAR() int64     { return s.GAR }
func (s *socket) SetGAR(v int64)    { s.GAR = v }
func (s *socket) UpdateGAR(v int64) { s.GAR = max64(s.GAR, v) }

func max64(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

func max32(x, y int32) int32 {
	if x > y {
		return x
	}
	return y
}

func min32(x, y int32) int32 {
	if x < y {
		return x
	}
	return y
}

// TODO: Address the last paragraph of Section 7.5.1 regarding SWL,AWL calculation

func (s *socket) SetSWAF(v int64) { s.SWAF = v }
func (s *socket) SetSWBF(v int64) { s.SWBF = v }

// GetSWLH() computes SWL and SWH, see Section 7.5.1
func (s *socket) GetSWLH() (SWL int64, SWH int64) {
	return max64(s.GSR+1-s.SWBF/4, s.ISR), s.GSR + (3*s.SWBF)/4
}

// GetAWLH() computes AWL and AWH, see Section 7.5.1
func (s *socket) GetAWLH() (AWL int64, AWH int64) {
	return max64(s.GSS+1-s.SWAF, s.ISS), s.GSS
}

func (s *socket) InAckWindow(x int64) bool {
	awl, awh := s.GetAWLH()
	return awl <= x && x <= awh
}
