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
	ISS uint64 // Initial Sequence number Sent
	ISR uint64 // Initial Sequence number Received

	OSR uint64 // First OPEN Sequence number Received

	// Here and elsewhere, "greatest" is measured in circular sequence space (modulo 2^48)
	GSS uint64 // Greatest Sequence number Sent

	GSR uint64 // Greatest valid Sequence number Received (consequently, sent as AckNo back)
	GAR uint64 // Greatest valid Acknowledgement number Received on a non-Sync; initialized to S.ISS

	CCIDA byte // CCID in use for the A-to-B half-connection, Section 10
	CCIDB byte // CCID in use for the B-to-A half-connection, Section 10

	SWAF uint64 // Sequence Window/A Feature, see Section 7.5.1
	SWBF uint64 // Sequence Window/B Feature, see Section 7.5.1

	State       int
	Server      bool   // True if the endpoint is a server, false if it is a client
	ServiceCode uint32 // The service code of this connection

	PMTU  uint32 // Path Maximum Transmission Unit
	CCMPS uint32 // Congestion Control Maximum Packet Size

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
	RTT_DEFAULT            = 2e8      // 0.2 sec, default Round-Trip Time when no measurement is available
	MSL                    = 2 * 60e9 // 2 mins in nanoseconds, Maximum Segment Lifetime, Section 3.4
	PARTOPEN_BACKOFF_FIRST = 200e6    // 200 miliseconds in nanoseconds, Section 8.1.5
	PARTOPEN_BACKOFF_MAX   = 4 * MSL  // 8 mins in nanoseconds, Section 8.1.5
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

func (s *socket) GetMPS() uint32 { return minUint32(s.CCMPS, s.PMTU) }

func (s *socket) GetPMTU() uint32  { return s.PMTU }
func (s *socket) SetPMTU(v uint32) { s.PMTU = v }

func (s *socket) GetCCMPS() uint32  { return s.CCMPS }
func (s *socket) SetCCMPS(v uint32) { s.CCMPS = v }

func (s *socket) GetRTT() int64  { return s.RTT }
func (s *socket) SetRTT(v int64) { s.RTT = v }

func (s *socket) SetServer(v bool) { s.Server = v }
func (s *socket) IsServer() bool   { return s.Server }

func (s *socket) GetState() int  { return s.State }
func (s *socket) SetState(v int) { s.State = v }

func (s *socket) SetServiceCode(v uint32) { s.ServiceCode = v }
func (s *socket) GetServiceCode() uint32  { return s.ServiceCode }

// ChooseISS chooses a safe Initial Sequence Number
func (s *socket) ChooseISS() uint64 {
	iss := uint64(rand.Int63()) & 0xffffff
	s.ISS = iss
	return iss
}

func (s *socket) SetISR(v uint64) { s.ISR = v }

func (s *socket) GetOSR() uint64  { return s.OSR }
func (s *socket) SetOSR(v uint64) { s.OSR = v }

func (s *socket) GetGSS() uint64  { return s.GSS }
func (s *socket) SetGSS(v uint64) { s.GSS = v }

func (s *socket) GetGSR() uint64     { return s.GSR }
func (s *socket) SetGSR(v uint64)    { s.GSR = v }
func (s *socket) UpdateGSR(v uint64) { s.GSR = maxu64(s.GSR, v) }

func (s *socket) GetGAR() uint64     { return s.GAR }
func (s *socket) SetGAR(v uint64)    { s.GAR = v }
func (s *socket) UpdateGAR(v uint64) { s.GAR = maxu64(s.GAR, v) }

func maxu64(x, y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}

// TODO: Address the last paragraph of Section 7.5.1 regarding SWL,AWL calculation

func (s *socket) SetSWABF(swaf, swbf uint64) {
	s.SWAF, s.SWBF = swaf, swbf
}

// GetSWLH() computes SWL and SWH, see Section 7.5.1
func (s *socket) GetSWLH() (SWL uint64, SWH uint64) {
	return maxu64(s.GSR+1-s.SWBF/4, s.ISR), s.GSR + (3*s.SWBF)/4
}

// GetAWLH() computes AWL and AWH, see Section 7.5.1
func (s *socket) GetAWLH() (AWL uint64, AWH uint64) {
	return maxu64(s.GSS+1-s.SWAF, s.ISS), s.GSS
}

func (s *socket) InAckWindow(x uint64) bool {
	awl, awh := s.GetAWLH()
	return awl <= x && x <= awh
}
