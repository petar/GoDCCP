// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "rand"

// socket is a data structure, maintaining the DCCP socket variables.
// socket's methods are not re-entrant
type socket struct {

	ISS	uint64	// Initial Sequence number Sent
	ISR	uint64	// Initial Sequence number Received

	OSR	uint64	// First OPEN Sequence number Received

	// Here and elsewhere, "greatest" is measured in circular sequence space (modulo 2^48)
	GSS	uint64	// Greatest Sequence number Sent

	GSR	uint64	// Greatest valid Sequence number Received (consequently, sent as AckNo back)
	GAR	uint64	// Greatest valid Acknowledgement number Received on a non-Sync; initialized to S.ISS

	SWBF	uint64	// Sequence Window/B Feature, see Section 7.5.1
	SWAF	uint64	// Sequence Window/A Feature, see Section 7.5.1

	State	int
}

const (
	MSL = 2*60e9	// 2 mins in nanoseconds
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

func (s *socket) GetMSL() int64 { return MSL }

func (s *socket) GetState() int { return s.State }

func (s *socket) SetState(v int) { s.State = v }

// ChooseISS chooses a safe Initial Sequence Number
func (s *socket) ChooseISS() uint64 { 
	iss := uint64(rand.Int63()) & 0xffffff 
	s.ISS = iss
	return iss
}

func (s *socket) SetISR(v uint64) { s.ISR = v }

func (s *socket) GetOSR() uint64 { return s.OSR }
func (s *socket) SetOSR(v uint64) { s.OSR = v }

func (s *socket) GetGSS() uint64 { return s.GSS }
func (s *socket) SetGSS(v uint64) { s.GSS = v }

func (s *socket) GetGSR() uint64 { return s.GSR }
func (s *socket) SetGSR(v uint64) { s.GSR = v }

func (s *socket) GetGAR() uint64 { return s.GAR }
func (s *socket) SetGAR(v uint64) { s.GAR = v }

func maxu64(x,y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}

// TODO: Address the last paragraph of Section 7.5.1 regarding SWL,AWL calculation

// GetSWL_SWH() computes SWL and SWH, see Section 7.5.1
func (s *socket) GetSWL_SWH() (SWL uint64, SWH uint64) {
	return maxu64(s.GSR + 1 - s.SWBF/4, s.ISR), s.GSR + (3*s.SWBF)/4
}

// GetAWL_AWH() computes AWL and AWH, see Section 7.5.1
func (s *socket) GetAWL_AWH() (AWL uint64, AWH uint64) {
	return maxu64(s.GSS + 1 - s.SWAF, s.ISS), s.GSS
}

func (s *socket) InAckWindow(x uint64) bool {
	awl, awh := s.GetAWL_AWH()
	return awl <= x && x <= awh
}
