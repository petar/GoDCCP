// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"testing"
)

func newTestJoiner(name string) *testJoiner {
	return &testJoiner{
		name:     name,
		joinChan: make(chan bool),
	}
}

type testJoiner struct {
	name     string
	joinChan chan bool
}

func (tj *testJoiner) Join() {
	_, _ = <-tj.joinChan
}

// indicate that this joiner is complete and ready to Join()
func (tj *testJoiner) Complete() {
	close(tj.joinChan)
}

func (tj *testJoiner) String() string {
	return tj.name
}

func TestMultipleJoin(t *testing.T) {
	test := NewGoJoin("testJoiner")
	joiner := newTestJoiner("testJoiner")
	test.Add(joiner)
	go func(){joiner.Complete()}()
	test.Join()
	test.Join()
}

func TestAddAfterJoin(t *testing.T) {
	test := NewGoJoin("testJoiner")
	joiner := newTestJoiner("initialJoiner")
	test.Add(joiner)
	go func(){joiner.Complete()}()
	test.Join()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("should have paniced")
		}
	}()
	test.Add(newTestJoiner("secondJoiner"))
}

func TestGoJoinSingle(t *testing.T) {
	joiner := newTestJoiner("testJoiner")

	test := NewGoJoin("test", joiner)
	res := make(chan string)
	go func() {
		test.Join()
		res <- "groupJoined"
	}()
	go func() {
		joiner.Complete()
		res <- "joinerComplete"
	}()

	first := <-res
	second := <-res

	if first != "joinerComplete" {
		t.Fail()
	}

	if second != "groupJoined" {
		t.Fail()
	}
}

func TestGoJoinMany(t *testing.T) {
	var joiners []*testJoiner
	test := NewGoJoin("test")
	res := make(chan string)

	// create 100 test joiners
	for i := 0; i < 30; i++ {
		joiners = append(joiners, newTestJoiner(fmt.Sprintf("testjoiner: %v\n", i)))
	}

	// start adding all joiners,
	for i, joiner := range joiners {
		test.Add(joiner)
		// start trying Join() at half way
		if i == len(joiners)/2 {
			go func() {
				test.Join()
				res <- "groupJoined"
			}()
		}
		// mark every 2nd one as complete already, while we are still adding.
		if i%2 == 0 {
			j := joiner
			go func() {
				j.Complete()
				res <- "joinerComplete"
			}()
			joiners[i] = nil
		}
	}

	// signal all remaining joiners that they are allowed to join now.
	for _, joiner := range joiners {
		if joiner != nil {
			j := joiner
			go func() {
				j.Complete()
				res <- "joinerComplete"
			}()
		}
	}

	// we should see that all joiners end first...
	for i := 0; i < len(joiners); i++ {
		next := <-res
		if next != "joinerComplete" {
			t.Fail()
		}
	}
	// .. end then the group ends
	if <-res != "groupJoined" {
		t.Fail()
	}

}
