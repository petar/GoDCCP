// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"testing"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/ccid3"
)

const (
	lossDuration     = 10e9        // Duration of the experiment in ns
	lossSendRate     = 23          // Fixed sender rate in pps
	lossTransmitRate = 20          // Fixed transmission rate of the network in pps
)

// TestLoss checks that loss estimation matches actual
func TestLoss(t *testing.T) {

	run, plex := NewRuntime("loss")
	plex.HighlightSamples(??)

	clientConn, serverConn, clientToServer, _ := NewClientServerPipe(run)

	cargo := []byte{1, 2, 3}
	buf := make([]byte, len(cargo))

	// In order to force packet loss, we fix the send rate slightly above the
	// the pipeline rate.
	clientConn.Amb().Flags().SetUint32("FixRate", lossSendRate)
	serverConn.Amb().Flags().SetUint32("FixRate", lossSendRate)
	clientToServer.SetWriteRate(1e9, lossTransmitRate)

	cchan := make(chan int, 1)
	go func() {
		t0 := run.Now()
		for run.Now() - t0 < lossDuration {
			err := clientConn.Write(buf)
			if err != nil {
				break
			}
		}
		clientConn.Close()
		close(cchan)
	}()

	schan := make(chan int, 1)
	go func() {
		for {
			_, err := serverConn.Read()
			if err != nil {
				break
			}
		}
		close(schan)
	}()

	_, _ = <-cchan
	_, _ = <-schan

	// Shutdown the connections properly
	clientConn.Abort()
	serverConn.Abort()
	dccp.NewGoConjunction("end-of-test", clientConn.Waiter(), serverConn.Waiter()).Wait()
	dccp.NewAmb("line", run).E(dccp.EventMatch, "Server and client done.")
	if err := run.Close(); err != nil {
		t.Errorf("error closing runtime (%s)", err)
	}
}
