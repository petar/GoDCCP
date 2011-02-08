// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"


// Therefore, DCCP senders and receivers SHOULD reset their congestion state --
// essentially restarting congestion control from "slow start" or equivalent --
// on significant changes in the end-to-end path.

type CongestionControl interface {
	GetMPS() uint32		// Maximum Packet Size, CCMPS. Generally PMTU <= CCMPS
}
