// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"github.com/petar/GoDCCP/dccp"
)

// GuzzlePlex is a dccp.Guzzle that replicates Guzzle method invocations to a set of Guzzles
type GuzzlePlex struct {
	guzzles []dccp.Guzzle
}

func NewGuzzlePlex(guzzles ...dccp.Guzzle) *GuzzlePlex {
	return &GuzzlePlex{
		guzzles: guzzles,
	}
}

func (t *GuzzlePlex) Write(r *dccp.LogRecord) {
	for _, g := range t.guzzles {
		g.Write(r)
	}
}

func (t *GuzzlePlex) Sync() error {
	var err error
	for _, g := range t.guzzles {
		e := g.Sync()
		if err != nil {
			err = e
		}
	}
	return err
}

func (t *GuzzlePlex) Close() error {
	var err error
	for _, g := range t.guzzles {
		e := g.Close()
		if err != nil {
			err = e
		}
	}
	return err
}
