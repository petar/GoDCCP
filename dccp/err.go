// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

// TODO: Annotate each error with the circumstances that can cause it
var (
	ErrAlign         = os.NewError("align")
	ErrSize          = os.NewError("size")
	ErrSemantic      = os.NewError("semantic")
	ErrSyntax        = os.NewError("syntax")
	ErrNumeric       = os.NewError("numeric")
	ErrOption        = os.NewError("option")
	ErrOptionsTooBig = os.NewError("options too big")
	ErrOversize      = os.NewError("over size")
	ErrCsCov         = os.NewError("cscov")
	ErrChecksum      = os.NewError("checksum")
	ErrIPFormat      = os.NewError("ip format")
	ErrUnknownType   = os.NewError("unknown packet type")
	ErrUnsupported   = os.NewError("unsupported")
	ErrProto         = os.NewError("protocol error")
	ErrDrop          = os.NewError("dropped")
	ErrReset         = os.NewError("reset")
)
