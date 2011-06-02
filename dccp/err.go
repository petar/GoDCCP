// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

// ProtoError is a type that wraps all DCCP-specific errors.
// It is utilized to distinguish these errors from others, using type checks.
type ProtoError string

func (e ProtoError) String() string  { return string(e) }

func NewError(s string) os.Error { return ProtoError(s) }

// TODO: Annotate each error with the circumstances that can cause it
var (
	ErrAlign         = NewError("align")
	ErrSize          = NewError("size")
	ErrSemantic      = NewError("semantic")
	ErrSyntax        = NewError("syntax")
	ErrNumeric       = NewError("numeric")
	ErrOption        = NewError("option")
	ErrOptionsTooBig = NewError("options too big")
	ErrOversize      = NewError("over size")
	ErrCsCov         = NewError("cscov")
	ErrChecksum      = NewError("checksum")
	ErrIPFormat      = NewError("ip format")
	ErrUnknownType   = NewError("unknown packet type")
	ErrUnsupported   = NewError("unsupported")
	ErrProto         = NewError("protocol error")
	ErrDrop          = NewError("dropped")
	ErrReset         = NewError("reset")
	ErrTooBig        = NewError("too big")
	ErrTimeout       = NewError("timeout")
	ErrOverflow      = NewError("overflow")
)
