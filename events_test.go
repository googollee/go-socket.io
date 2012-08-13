package socketio

import (
	"testing"
)

type abc struct {
}

func TestEventEmitterEmit(t *testing.T) {
	ee := NewEventEmitter()
	c := make(chan int)
	err := ee.On("hello", func(name string, n int) {
		c <- n
	})
	if err != nil {
		t.Fatal(err)
	}
	ee.emit("hello", nil, 1)
	ee.emit("hello", nil, 2)
	n := <-c
	if n != 1 {
		t.Fatalf("expert %d got %d", 1, n)
	}
	n = <-c
	if n != 2 {
		t.Fatalf("expert %d got %d", 2, n)
	}
}

func TestEventEmitterEmitRaw(t *testing.T) {
	ee := NewEventEmitter()
	c := make(chan int)
	err := ee.On("hello", func(name string, n int) {
		c <- n
	})
	if err != nil {
		t.Fatal(err)
	}
	ee.emitRaw("hello", nil, []byte("[1]"))
	n := <-c
	if n != 1 {
		t.Fatalf("expert %d got %d", 1, n)
	}
}

func TestEventEmitterEmitCallback(t *testing.T) {
	ee := NewEventEmitter()
	err := ee.On("hello", func(name string, n int) int {
		return 2
	})
	if err != nil {
		t.Fatal(err)
	}
	c := make(chan []interface{})
	callback := func(args []interface{}) {
		c <- args
	}
	ee.emit("hello", callback, 1)
	got := <-c
	expect := []interface{}{2}
	if got[0] != expect[0] {
		t.Fatalf("expert %v got %v", expect, got)
	}
}
