// CCID7 is the name we pick for our implementation of a congestion control 
// protocol over DCCP. CCID7 is described by:
//      (*) Only extended numbers are used
//      (*) No feature negitation mechanisms are implemented
//		(*) Send NDP Count feature is always ON
//		(*) Allow Short Sequence Numbers feature is always OFF

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

// XXX: In theory, long lived connections may wrap around the AckNo/SeqNo space
// in which case maxu64() should not be used below. This will never happen however
// if we are using 48-bit numbers exclusively

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
