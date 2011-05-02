// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"net"
	"testing"
	//"time"
)

// TODO: 
//   Test over-sized writes
//   Test that small writes are not combined in single packets

var done = make(chan int)

func RunAccepter(t *testing.T) {
	fmt.Printf("starting accepter...\n")
	defer fmt.Printf("ending accepter...\n")

	laddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:44000")
	if err != nil {
		t.Fatalf("udp addr %s", err)
	}
	link, err := BindUDPLink("udp", laddr)
	if err != nil {
		t.Fatalf("bind %s", err)
	}
	m := newMux(link, link.FragmentLen())
	for {
		c, err := m.Accept()
		if err != nil {
			t.Fatalf("accept %s", c, err)
		}
		go func() {
			fmt.Printf("accepted\n")
			p := make([]byte, 2000)
			n, err := c.Read(p)
			if err != nil {
				t.Fatalf("read: %s", err)
				return
			}
			if n != 4 {
				t.Fatalf("read size: %d != 4", n)
				return
			}
			i := decode4ByteUint(p[:4])
			fmt.Printf("got: %d\n", i)
			done <- 1
		}()
	}
}

func RunDialer(t *testing.T) {
	fmt.Printf("starting dialer...\n")
	defer fmt.Printf("ending dialer...\n")

	laddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:44001")
	if err != nil {
		t.Fatalf("d·udp·addr: %s", err)
	}
	link, err := BindUDPLink("udp", laddr)
	if err != nil {
		t.Fatalf("d·bind: %s", err)
	}
	m := newMux(link, link.FragmentLen())

	raddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:44000")
	if err != nil {
		t.Fatalf("d·udp·addr %s", err)
	}
	c, err := m.Dial(raddr)
	if err != nil {
		t.Fatalf("d·dial: %s", err)
	}
	p := make([]byte, 4)
	encode4ByteUint(7, p)
	n, err := c.Write(p)
	if err != nil {
		t.Fatalf("d·write: %s", err)
	}
	if n != 4 {
		t.Fatalf("d·write·size: %d != 4", n)
	}
	err = c.Close()
	if err != nil {
		t.Fatalf("d·close: %s", err)
	}
}

func TestMux(t *testing.T) {
	InstallCtrlCPanic()
	go RunAccepter(t)
	go RunDialer(t)
	<-done
}
