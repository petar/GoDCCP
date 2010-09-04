// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

func ReadGenericHeader(buf []byte) (*GenericHeader, os.Error) {
	if len(buf) < 12 {
		return nil, ErrSize
	}
	gh := &GenericHeader{}
	eat := buf

	gh.SourcePort = decode2ByteUint(eat[0:2])
	eat = eat[2:]

	gh.DestPort = decode2ByteUint(eat[0:2])
	eat = eat[2:]

	gh.DataOffset = decode1ByteUint(eat[0:1])
	eat = eat[1:]
	? // check bounds

	gh.CCVal = eat[0] >> 4
	gh.CsCov = eat[0] & 0x0f
	eat = eat[1:]
	? // check bounds

	? // XXX checksum?

	?
}

