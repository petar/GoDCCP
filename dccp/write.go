// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	//"fmt"
	"os"
)

// getFootprint() retutns the option's wire footprint, which includes
// a preceding Mandatory option on the wire, if necessary
func (opt *Option) getFootprint() (int, os.Error) {
	if opt.Type == OptionPadding || opt.Type == OptionMandatory {
		return 0, ErrOption
	}
	if isOptionSingleByte(opt.Type) {
		if opt.Data != nil && len(opt.Data) > 0 {
			return 0, ErrOption
		}
		if opt.Mandatory {
			return 2, nil
		}
		return 1, nil
	}
	mopt := 0
	if opt.Mandatory {
		mopt = 1
	}
	if opt.Data == nil {
		return mopt + 2, nil // Option Type byte + Option Len byte
	}
	if len(opt.Data) > (255 - 2) {
		return 0, ErrOption
	}
	return mopt + 2 + len(opt.Data), nil
}

// getOptionsFootprint() returns the size of the options part of the header,
// while not including any space for options whose type is not compatible with
// the type of the header
func (gh *Header) getOptionsFootprint() (int, os.Error) {
	if gh.Options == nil {
		return 0, nil
	}
	r := 0
	for _, opt := range gh.Options {
		if !isOptionValidForType(opt.Type, gh.Type) {
			if opt.Mandatory {
				return 0, ErrOption
			}
			continue
		}
		s, err := opt.getFootprint()
		if err != nil {
			return 0, err
		}
		r += s
	}
	if r%4 != 0 {
		r += 4 - (r % 4)
	}
	return r, nil
}

// getHeaderFootprint() returns the size of the wire-format packet header (excluding app data)
func (gh *Header) getHeaderFootprint(allowShortSeqNoFeature bool) (int, os.Error) {

	// Check that X and Type are compatible
	if !areTypeAndXCompatible(gh.Type, gh.X, allowShortSeqNoFeature) {
		return 0, ErrSemantic
	}

	// Calculate the fixed-size portion of the header including (1a), (1b) and (1c)
	r := getFixedHeaderSize(gh.Type, gh.X)

	// Add any size needed for (2), options and padding
	optsFoot, err := gh.getOptionsFootprint()
	if err != nil {
		return 0, err
	}
	r += optsFoot

	if 255*4 < r {
		return 0, ErrOversize
	}

	return r, nil
}

// Write() writes the DCCP header to two return buffers.
// The first one is the header part, and the second one is the data
// part which simply equals the slice Header.Data
func (gh *Header) Write(sourceIP, destIP []byte,
	protoNo byte,
	allowShortSeqNoFeature bool) (header []byte, err os.Error) {

	err = verifyIPAndProto(sourceIP, destIP, protoNo)
	if err != nil {
		return nil, err
	}

	dataOffset, err := gh.getHeaderFootprint(allowShortSeqNoFeature)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, dataOffset+len(gh.Data))

	k := 0

	// Write (1a) Generic Header
	Encode2ByteUint(gh.SourcePort, buf[k:k+2])
	k += 2

	Encode2ByteUint(gh.DestPort, buf[k:k+2])
	k += 2

	// Write app data offset
	Encode1ByteUint(byte(dataOffset>>2), buf[k:k+1])
	k += 1

	// Write CCVal
	buf[k] = gh.CCVal << 4

	// Write CsCov
	buf[k] |= gh.CsCov & 0x0f
	k += 1

	// Write 0 in the Checksum field
	buf[k], buf[k+1] = 0, 0
	k += 2 // Skip over the checksum field now. It is filled in at the end.

	// Write Res and Type
	buf[k] = (gh.Type & 0x0f) << 1

	// Write X
	if gh.X {
		buf[k] |= 0x01
	}
	k += 1

	// Write SeqNo
	switch gh.X {
	case false:
		Encode3ByteUint(uint32(gh.SeqNo), buf[k:k+3])
		k += 3
	case true:
		buf[k] = 0
		k += 1 // skip over Reserved
		if gh.SeqNo < 0 {
			panic("seqno < 0")
		}
		Encode6ByteUint(uint64(gh.SeqNo), buf[k:k+6])
		k += 6
	}

	// Write (1b) Acknowledgement Number Subheader

	switch getAckNoSubheaderSize(gh.Type, gh.X) {
	case 0:
	case 4:
		buf[k] = 0
		k += 1 // Skip over Reserved
		Encode3ByteUint(uint32(gh.AckNo), buf[k:k+3])
		k += 3
	case 8:
		buf[k], buf[k+1] = 0, 0
		k += 2 // Skip over Reserved
		if gh.AckNo < 0 {
			panic("ackno < 0")
		}
		Encode6ByteUint(uint64(gh.AckNo), buf[k:k+6])
		k += 6
	default:
		panic("unreach")
	}

	// Write (1c) Code Subheader: Service Code, or Reset Code and Reset Data fields
	switch gh.Type {
	case Request, Response:
		Encode4ByteUint(gh.ServiceCode, buf[k:k+4])
		k += 4
	case Reset:
		buf[k] = gh.ResetCode
		n := copy(buf[k+1:k+4], gh.ResetData)
		for i := 0; i < 3-n; i++ {
			buf[k+1+n+i] = 0
		}
		k += 4
	}

	// Write (2) Options and Padding
	writeOptions(gh.Options, buf[k:dataOffset], gh.Type)

	// Write checksum
	dlen := len(gh.Data)
	appCov, err := getChecksumAppCoverage(gh.CsCov, dlen)
	if err != nil {
		return nil, err
	}
	csum := csumSum(buf[0:dataOffset])
	csum = csumAdd(csum, csumPseudoIP(sourceIP, destIP, protoNo, len(buf)))
	if appCov > 0 {
		csum = csumAdd(csum, csumSum(gh.Data[0:appCov]))
	}
	csum = csumDone(csum)
	csumUint16ToBytes(csum, buf[6:8])

	// Write data
	copy(buf[dataOffset:], gh.Data)

	return buf, nil
}

func writeOptions(opts []*Option, buf []byte, Type byte) {
	if len(buf) & 0x3 != 0 {
		panic("logic")
	}
	k := 0
	for _, opt := range opts {
		if !isOptionValidForType(opt.Type, Type) {
			continue
		}
		if opt.Mandatory {
			buf[k] = OptionMandatory
			k++
		}
		buf[k] = opt.Type
		k++
		if isOptionSingleByte(opt.Type) {
			continue
		}
		if opt.Data == nil {
			buf[k] = 2
			k++
			continue
		}
		buf[k] = byte(2 + len(opt.Data))
		k++
		n := copy(buf[k:], opt.Data)
		if n != len(opt.Data) {
			panic("opt data len")
		}
		k += len(opt.Data)
	}
	if len(buf)-k >= 4 {
		panic("opt padding len")
	}
	for i := 0; i < len(buf)-k; i++ {
		buf[k+i] = 0
	}
}
