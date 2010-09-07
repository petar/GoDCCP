// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

func ReadGenericHeader(buf []byte) (*GenericHeader, os.Error) {
	if len(buf) < 12 {
		return nil, ErrSize
	}
	gh := &GenericHeader{}
	k := 0

	// Read (1) Generic Header

	gh.SourcePort = decode2ByteUint(buf[k:k+2])
	k += 2

	gh.DestPort = decode2ByteUint(buf[k:k+2])
	k += 2

	// Compute the Data Offset in bytes
	dataOffset := int(decode1ByteUint(buf[k:k+1])) * wireWordSize
	k += 1

	gh.CCVal = buf[k] >> 4
	gh.CsCov = buf[k] & 0x0f
	k += 1
	// XXX: CCVal/CsCov check bounds

	/* checksum := */ decode2ByteUint(buf[k:k+2])
	k += 2
	// XXX: Checksum check

	// gh.Res = buf[k] >> 5 // The 3-bit Res field should be ignored
	gh.Type = int((buf[k] >> 1) & 0x0f)
	gh.X = (buf[k] & 0x01) == 1
	k += 1

	// XXX: Don't assume that AllowShortSeqNoFeature is false
	if !areTypeAndXCompatible(gh.Type, gh.X, false) {
		return nil, ErrSemantic
	}
	
	// Check Data Offset bounds
	if dataOffset < calcFixedHeaderSize(gh.Type, gh.X) || dataOffset > len(buf) {
		return nil, ErrNumeric
	}

	switch gh.X {
	case false:
		gh.SeqNo = uint64(decode3ByteUint(buf[k:k+3]))
		k += 3
	case true:
		padding := decode1ByteUint(buf[k:k+1])
		k += 1
		if padding != 0 {
			return nil, ErrNumeric
		}
		gh.SeqNo = decode6ByteUint(buf[k:k+6])
		k += 6
	}

	// Read (2) Acknowledgement Number Subheader

	switch calcAckNoSubheaderSize(gh.Type, gh.X) {
	case 0:
	case 4:
		padding := decode1ByteUint(buf[k:k+1])
		k += 1
		if padding != 0 {
			return nil, ErrNumeric
		}
		gh.AckNo = uint64(decode3ByteUint(buf[k:k+3]))
		k += 3
	case 8:
		padding := decode2ByteUint(buf[k:k+2])
		k += 2
		if padding != 0 {
			return nil, ErrNumeric
		}
		gh.AckNo = decode6ByteUint(buf[k:k+6])
		k += 6
	default:
		panic("unreach")
	}

	// Read (3) Code Subheader: Service Code, or Reset Code and Reset Data fields
	switch gh.Type {
	case Request, Response:
		gh.ServiceCode = decode6ByteUint(buf[k:k+4])
		k += 4
	case Reset:
		gh.Reset = buf[k:k+4]
		k += 4
	}

	// Read (4) Options and Padding
	opts, err := readOptions(buf[k:dataOffset])
	if err != nil {
		return nil, err
	}
	opts, err = sanitizeOptions(gh.Type, opts)
	if err != nil {
		return nil, err
	}

	// Read (5) Application Data
	gh.Data = buf[dataOffset:]

	return gh, nil
}

func readOptions(buf []byte) ([]Option, os.Error) {
	if len(buf) >> 2 != 0 {
		return nil, ErrAlign
	}

	opts := make([]Option, len(buf))
	j, k := 0, 0
	for k < len(buf) {
		// Read option type
		t := int(buf[k])
		k += 1

		if isOptionSingleByte(t) {
			opts[j].Type = t
			opts[j].Data = make([]byte, 0)
			j += 1
			continue
		}

		// Read option length
		if k+1 > len(buf) {
			break
		}
		l := int(buf[k])
		k += 1
		if l < 2 || k+l-2 > len(buf) {
			break
		}
		
		opts[j].Type = t
		opts[j].Data = buf[k:k+l-2]
		k += l-2
		j += 1

	}
	
	return opts[0:j], nil
}

func sanitizeOptions(Type int, opts []Option) ([]Option, os.Error) {
	r := make([]Option, len(opts))
	j := 0

	nextIsMandatory := false
	for i := 0; i < len(opts); i++ {
		if !isOptionValidForType(opts[i].Type, Type) {
			if nextIsMandatory {
				return nil, ErrOption
			}
			nextIsMandatory = false
			continue
		}
		switch opts[i].Type {
		case OptionMandatory:
			if nextIsMandatory {
				return nil, ErrOption
			}
			nextIsMandatory = true
		case OptionPadding:
			nextIsMandatory = false
			continue
		default:
			r[j] = opts[i]
			if nextIsMandatory {
				r[j].Mandatory = true
				nextIsMandatory = false
			}
			j++
			continue
		}
	}
	if nextIsMandatory {
		return nil, ErrOption
	}

	return r[0:j], nil
}
