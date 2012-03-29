// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

var lk sync.Mutex

func GoRoutineID() {
}

func PrintGoID(delay time.Duration, c chan int) {
	lk.Lock()
	//debug.PrintStack()
	fmt.Println(string(debug.Stack()))
	fmt.Printf("————\n\n")
	lk.Unlock()
	//fmt.Printf("This goroutine %v\n", GoRoutineID())
	c <- 1
}

func main() {
	c := make(chan int)
	go PrintGoID(time.Second, c)
	go PrintGoID(time.Second*2, c)
	go PrintGoID(time.Second*3, c)
	<-c; <-c; <-c
}
