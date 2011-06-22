// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package retransmit

import (
	"io"
	"os"
	"github.com/petar/GoDCCP/dccp"
)

func NewRetransmit(bc dccp.BlockConn) io.ReadWriteCloser {
	c := &conn{
		bc:        bc,
		readFirst: 0,
		readWin:   make([][]byte, RETRANSMIT_WIDTH),
		readChan:  make(chan []byte),
	}
	go c.readLoop()
	go c.writeLoop()
	return c
}

type conn struct {
	bc dccp.BlockConn

	// Read fields
	readLock dccp.Mutex
	readTail []byte

	// readLoop fields
	readFirst uint32
	readWin   [][]byte
	readChan  chan []byte

	// Write fields
	writeLock  dccp.Mutex
	writeFirst uint32
	writeWin   [][]byte
	writeChan  chan []byte
}

const RETRANSMIT_WIDTH = 128 // The retransmit window width must be a multiple of 8

func (c *conn) Read(p []byte) (n int, err os.Error) {
	c.readLock.Lock()
	defer c.readLock.Unlock()

	for len(p) > 0 {
		// Read incoming data
		if len(c.readTail) == 0 {
			b, ok := <-c.readChan
			if !ok {
				return n, os.EOF
			}
			c.readTail = b
		}

		// Copy to user buffer
		k := copy(p, c.readTail)
		c.readTail = c.readTail[k:]
		p = p[k:]
		n += k
	}

	return n, nil
}

func makeAckMap(win [][]byte, first uint32) []byte {
	am := make([]byte, len(win) / 8)
	for i, b := range win {
		if b == nil {
			am[i/8] |= (1 << uint(i % 8))
		}
	}
	return am
}

func (c *conn) readLoop() {
	for {
		b, err := c.bc.ReadBlock()
		if err != nil {
			break
		}
		h, err := readHeader(b)
		if err != nil {
			break
		}

		?
	}
	??
}

func (c *conn) Write(p []byte) (n int, err os.Error) {
	panic("")
}

func (c *conn) writeLoop() {
	panic("")
}

func (c *conn) Close() os.Error {
	panic("")
}
