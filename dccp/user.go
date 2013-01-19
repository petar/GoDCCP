// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
)

// This is an approximate upper bound on the size of options that are
// allowed on a Data or DataAck packet. See isOptionValidForType.
const maxDataOptionSize = 24

// GetMTU() returns the maximum size of an application-level data block that can be passed
// to Write This is an informative number. Packets are sent anyway, but they may be
// dropped by the link layer or a router.
func (c *Conn) GetMTU() int {
	c.Lock()
	defer c.Unlock()
	c.syncWithLink()
	return int(c.socket.GetMPS()) - maxDataOptionSize - getFixedHeaderSize(DataAck, true)
}

// Write blocks until the slice b is sent.
func (c *Conn) Write(data []byte) error {

	//?

	c.writeDataLk.Lock()
	defer c.writeDataLk.Unlock()
	if c.writeData == nil {
		return ErrBad
	}
	c.writeData <- data
	return nil
}

// Read blocks until the next packet of application data is received. Successfuly read data
// is returned in a slice. The error returned by Read behaves according to io.Reader. If the
// connection was never established or was aborted, Read returns ErrIO. If the connection
// was closed normally, Read returns io.EOF. In the event of a non-nil error, successive
// calls to Read return the same error.
func (c *Conn) Read() (b []byte, err error) {
	c.readAppLk.Lock()
	readApp := c.readApp
	c.readAppLk.Unlock()
	if readApp == nil {
		if c.Error() == nil {
			panic("torn connection missing error")
		}
		return nil, c.Error()
	}
	b, ok := <-readApp
	if !ok {
		if c.Error() == nil {
			panic("torn connection missing error")
		}
		// The connection has been closed
		return nil, c.Error()
	}
	return b, nil
}

func (c *Conn) Error() error {
	c.Lock()
	defer c.Unlock()
	return c.err
}

// Close implements SegmentConn.Close.
// It closes the connection, Section 8.3.
func (c *Conn) Close() error {
	c.Lock()
	defer c.Unlock()
	state := c.socket.GetState()
	switch state {
	case LISTEN:
		c.reset(ResetClosed, ErrEOF)
		return nil
	case REQUEST:
		c.reset(ResetClosed, ErrEOF)
		return nil
	case RESPOND:
		c.reset(ResetClosed, ErrEOF)
	case PARTOPEN, OPEN:
		c.inject(c.generateClose())
		c.gotoCLOSING()
		return nil
	case CLOSEREQ, CLOSING, TIMEWAIT, CLOSED:
		if c.err == nil {
			panic(fmt.Sprintf("%s without error", StateString(state)))
		}
		return c.err
	}
	panic("unknown state")
}

func (c *Conn) Abort() {
	c.abortWith(ResetAborted)
}

// LocalLabel implements SegmentConn.LocalLabel
func (c *Conn) LocalLabel() Bytes { return c.hc.LocalLabel() }

// RemoteLabel implements SegmentConn.RemoteLabel
func (c *Conn) RemoteLabel() Bytes { return c.hc.RemoteLabel() }

// SetReadExpire implements SegmentConn.SetReadExpire
func (c *Conn) SetReadExpire(nsec int64) error {
	panic("not implemented")
}
