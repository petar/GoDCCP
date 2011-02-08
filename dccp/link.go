// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp


type Link interface {

	func MTU() uint32	// (Path) Maximum Transmission Unit, PMTU
}
