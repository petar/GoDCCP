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

	// Read (1) Generic Header

	gh.SourcePort = decode2ByteUint(eat[0:2])
	eat = eat[2:]

	gh.DestPort = decode2ByteUint(eat[0:2])
	eat = eat[2:]

	// Compute the Data Offset in bytes
	dataOffset := int(decode1ByteUint(eat[0:1])) * wireWordSize
	eat = eat[1:]

	gh.CCVal = eat[0] >> 4
	gh.CsCov = eat[0] & 0x0f
	eat = eat[1:]
	? // CCVal/CsCov check bounds

	checksum := decode2ByteUint(eat[0:2])
	eat = eat[2:]
	? // Checksum check

	// gh.Res = eat[0] >> 5 // The 3-bit Res field should be ignored
	gh.Type = (eat[0] >> 1) & 0x0f
	gh.X = eat[0] & 0x01
	eat = eat[1:]

	// XXX: Don't assume that AllowShortSeqNoFeature is false
	if !areTypeAndXCompatible(gh.Type, gh.X, false) {
		return nil, ErrSemantic
	}
	
	// Check Data Offset bounds
	if dataOffset < calcFixedHeaderSize(Type, X) || dataOffset > len(buf) {
		return nil, ErrNumeric
	}

	switch X {
	case 0:
		gh.SeqNo = uint64(decode3ByteUint(eat[0:3]))
		eat = eat[3:]
	case 1:
		padding := decode1ByteUint(eat[0:1])
		eat = eat[1:]
		if padding != 0 {
			return nil, ErrNumeric
		}
		gh.SeqNo = decode6ByteUint(eat[0:6])
		eat = eat[6:]
	default:
		panic("unreach")
	}

	// Read (2) Acknowledgement Number Subheader

	switch calcAckNoSubheaderSize(gh.Type, gh.X) {
	case 0:
	case 4:
		padding := decode1ByteUint(eat[0:1])
		eat = eat[1:]
		if padding != 0 {
			return nil, ErrNumeric
		}
		gh.AckNo = decode3ByteUint(eat[0:3])
		eat = eat[3:]
	case 8:
		padding := decode2ByteUint(eat[0:2])
		eat = eat[2:]
		if padding != 0 {
			return nil, ErrNumeric
		}
		gh.AckNo = decode6ByteUint(eat[0:6])
		eat = eat[6:]
	default:
		panic("unreach")
	}

	// Read (3) Code Subheader: Service Code, or Reset Code and Reset Data fields

	// Read (4) Options and Padding

	// Read (5) Application Data

	?
}

