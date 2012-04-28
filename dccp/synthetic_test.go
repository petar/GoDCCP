package dccp

import (
	"fmt"
	"testing"
)

func A(r Runtime) {
	go func() { B(r) }()
	r.Sleep(6000)
	fmt.Printf("A, now=%d\n", r.Now())
}

func B(r Runtime) {
	r.Sleep(3000)
	fmt.Printf("B, now=%d\n", r.Now())
}

func TestGoSynthetic(t *testing.T) {
	GoSynthetic(A)
}

func TestNewSyntheticRuntime(t *testing.T) {
	runtime := NewSyntheticRuntime()
	go func() { A(runtime) }()
	runtime.Join()
}

func TestSleeperQueue(t *testing.T) {
	var sleepers sleeperQueue
	sleepers.Add(&scheduledToSleep{wake:3})
	sleepers.Add(&scheduledToSleep{wake:2})
	sleepers.Add(&scheduledToSleep{wake:1})
	if x := sleepers.DeleteMin(); x == nil || x.wake != 1 {
		t.Errorf("expecting 1")
	}
	if x := sleepers.DeleteMin(); x == nil || x.wake != 2 {
		t.Errorf("expecting 2")
	}
	if x := sleepers.DeleteMin(); x == nil || x.wake != 3 {
		t.Errorf("expecting 3")
	}
	if sleepers.Len() != 0 {
		t.Errorf("expecting 0-length")
	}
}
