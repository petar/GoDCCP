// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
	"fmt"
	"strconv"
)

func (h *Header) String() string {
	var w bytes.Buffer
	switch h.Type {
	case Request:
		fmt.Fprintf(&w, "T:%s X:%d SeqNo:%d SC:%d ··· SP:%2d DP:%2d",
			typeToString(h.Type), h.X, h.SeqNo, h.ServiceCode,
			h.SourcePort, h.DestPort)
	case Response:
		fmt.Fprintf(&w, "T:%s X:%d SeqNo:%d AckNo:%d SC:%d ··· SP:%2d DP:%2d",
			typeToString(h.Type), h.X, h.SeqNo, h.AckNo, h.ServiceCode,
			h.SourcePort, h.DestPort)
	case Data:
		fmt.Fprintf(&w, "T:%s X:%d SeqNo:%d ··· #D:%d",
			typeToString(h.Type), h.X, h.SeqNo,
			len(h.Data))
	case Ack:
		fmt.Fprintf(&w, "T:%s X:%d SeqNo:%d AckNo:%d",
			typeToString(h.Type), h.X, h.SeqNo, h.AckNo)
	case DataAck:
		fmt.Fprintf(&w, "T:%s X:%d SeqNo:%d AckNo:%d ··· #D:%d",
			typeToString(h.Type), h.X, h.SeqNo, h.AckNo,
			len(h.Data))
	case CloseReq:
		fmt.Fprintf(&w, "T:%s X:%d SeqNo:%d AckNo:%d",
			typeToString(h.Type), h.X, h.SeqNo, h.AckNo)
	case Close:
		fmt.Fprintf(&w, "T:%s X:%d SeqNo:%d AckNo:%d",
			typeToString(h.Type), h.X, h.SeqNo, h.AckNo)
	case Reset:
		fmt.Fprintf(&w, "T:%s X:%d SeqNo:%d AckNo:%d ··· RC: %s #RD:%d",
			typeToString(h.Type), h.X, h.SeqNo, h.AckNo,
			resetCodeToString(h.ResetCode), len(h.ResetData))
	case Sync:
		fmt.Fprintf(&w, "T:%s X:%d SeqNo:%d AckNo:%d",
			typeToString(h.Type), h.X, h.SeqNo, h.AckNo)
	case SyncAck:
		fmt.Fprintf(&w, "T:%s X:%d SeqNo:%d AckNo:%d",
			typeToString(h.Type), h.X, h.SeqNo, h.AckNo)
	default:
		panic("unknown packet type")
	}
	return string(w.Bytes())
}

func typeToString(typ byte) string {
	switch typ {
	case Request:
		return "Request"
	case Response:
		return "Response"
	case Data:
		return "Data"
	case Ack:
		return "Ack"
	case DataAck:
		return "DataAck"
	case CloseReq:
		return "CloseReq"
	case Close:
		return "Close"
	case Reset:
		return "Reset"
	case Sync:
		return "Sync"
	case SyncAck:
		return "SyncAck"
	}
	panic("un")
}

func resetCodeToString(resetCode byte) string {
	switch resetCode {
	case ResetUnspecified:
		return "Unspecified"
	case ResetClosed:
		return "Closed"
	case ResetAborted:
		return "Aborted"
	case ResetNoConnection:
		return "No Connection"
	case ResetPacketError:
		return "Packet Error"
	case ResetOptionError:
		return "Option Error"
	case ResetMandatoryError:
		return "Mandatory Error"
	case ResetConnectionRefused:
		return "Connection Refused"
	case ResetBadServiceCode:
		return "Bad Service Code"
	case ResetTooBusy:
		return "Too Busy"
	case ResetBadInitCookie:
		return "Bad Init Cookie"
	case ResetAgressionPenalty:
		return "Agression Penalty"
	}
	return strconv.Itoa(int(resetCode))
}
