// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

//"fmt"

// verifyIPAndProto() checks that both sourceIP# and destIP# are valid for protoNo#
func verifyIPAndProto(sourceIP, destIP []byte, protoNo byte) error {
	if sourceIP == nil || destIP == nil {
		return ErrIPFormat
	}
	if !((len(sourceIP) == 4 && len(destIP) == 4) || (len(sourceIP) == 16 && len(destIP) == 16)) {
		return ErrIPFormat
	}
	return nil
}

func ReadHeader(buf []byte,
	sourceIP, destIP []byte,
	protoNo byte,
	allowShortSeqNoFeature bool) (header *Header, err error) {

	err = verifyIPAndProto(sourceIP, destIP, protoNo)
	if err != nil {
		return nil, err
	}

	if len(buf) < 12 {
		return nil, ErrSize
	}
	gh := &Header{}
	k := 0

	// Read (1a) Generic Header

	gh.SourcePort = DecodeUint16(buf[k : k+2])
	k += 2

	gh.DestPort = DecodeUint16(buf[k : k+2])
	k += 2

	// Compute the Data Offset in bytes
	dataOffset := int(DecodeUint8(buf[k:k+1])) << 2
	k += 1

	// Read CCVal
	gh.CCVal = int8(buf[k] >> 4)

	// Read CsCov
	gh.CsCov = buf[k] & 0x0f
	k += 1

	k += 2 // Skip over the checksum field. It is implicitly used in checksum verification later

	// Read Res
	// gh.Res = buf[k] >> 5 // The 3-bit Res field should be ignored

	// Read Type
	gh.Type = (buf[k] >> 1) & 0x0f
	if !isTypeUnderstood(gh.Type) {
		return nil, ErrUnknownType
	}

	// Read X
	gh.X = (buf[k] & 0x01) == 1
	k += 1

	// Check that X and Type are compatible
	if !areTypeAndXCompatible(gh.Type, gh.X, allowShortSeqNoFeature) {
		return nil, ErrSemantic
	}

	// Check Data Offset bounds
	if dataOffset < getFixedHeaderSize(gh.Type, gh.X) || dataOffset > len(buf) {
		return nil, ErrNumeric
	}

	// Verify checksum
	appCov, err := getChecksumAppCoverage(gh.CsCov, len(buf)-dataOffset)
	if err != nil {
		return nil, err
	}
	csum := csumSum(buf[0:dataOffset])
	csum = csumAdd(csum, csumPseudoIP(sourceIP, destIP, protoNo, len(buf)))
	csum = csumAdd(csum, csumSum(buf[dataOffset:dataOffset+appCov]))
	csum = csumDone(csum)
	if csum != 0 {
		return nil, ErrChecksum
	}

	// Read SeqNo
	switch gh.X {
	case false:
		gh.SeqNo = int64(DecodeUint24(buf[k : k+3]))
		k += 3
	case true:
		padding := DecodeUint8(buf[k : k+1])
		k += 1
		if padding != 0 {
			return nil, ErrNumeric
		}
		gh.SeqNo = int64(DecodeUint48(buf[k : k+6]))
		k += 6
	}

	// Read (1b) Acknowledgement Number Subheader

	switch getAckNoSubheaderSize(gh.Type, gh.X) {
	case 0:
	case 4:
		padding := DecodeUint8(buf[k : k+1])
		k += 1
		if padding != 0 {
			return nil, ErrNumeric
		}
		gh.AckNo = int64(DecodeUint24(buf[k : k+3]))
		k += 3
	case 8:
		padding := DecodeUint16(buf[k : k+2])
		k += 2
		if padding != 0 {
			return nil, ErrNumeric
		}
		gh.AckNo = int64(DecodeUint48(buf[k : k+6]))
		k += 6
	default:
		panic("unreach")
	}

	// Read (1c) Code Subheader: Service Code, or Reset Code and Reset Data fields
	switch gh.Type {
	case Request, Response:
		gh.ServiceCode = DecodeUint32(buf[k : k+4])
		k += 4
	case Reset:
		gh.ResetCode = buf[k]
		gh.ResetData = buf[k+1 : k+4]
		k += 4
	}

	// Read (2) Options and Padding
	opts, err := readOptions(buf[k:dataOffset])
	if err != nil {
		return nil, err
	}
	opts, err = sanitizeOptionsAfterReading(gh.Type, opts)
	if err != nil {
		return nil, err
	}
	gh.Options = opts

	// Read (3) Application Data
	gh.Data = buf[dataOffset:]

	return gh, nil
}

func readOptions(buf []byte) ([]*Option, error) {
	if len(buf)&0x3 != 0 {
		return nil, ErrAlign
	}

	opts := make([]*Option, len(buf))
	j, k := 0, 0
	for k < len(buf) {
		o := &Option{}

		// Read option type
		t := buf[k]
		k += 1

		if isOptionSingleByte(t) {
			o.Type = t
			o.Data = make([]byte, 0)

			opts[j] = o
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

		o.Type = t
		o.Data = buf[k : k+l-2]
		k += l - 2

		opts[j] = o
		j += 1

	}

	return opts[0:j], nil
}

func sanitizeOptionsAfterReading(Type byte, opts []*Option) ([]*Option, error) {
	r := make([]*Option, len(opts))
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
