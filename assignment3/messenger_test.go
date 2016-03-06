package main

import "testing"

func TestSimple(t *testing.T) { // {{{1
    _, err := NewMsger(1, map[int]string {
        1: "tcp://127.0.0.1:1234",
        2: "tcp://127.0.0.1:2345",
        3: "tcp://127.0.0.1:3456",
    })
    if err != nil {
        t.Fatal("Creating messenger failed:", err)
    }
}
