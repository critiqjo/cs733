package main

import (
    //"github.com/critiqjo/cs733/assignment3/raft"
    //"reflect"
    "testing"
)

func initPster(t *testing.T, dbpath string) *SimplePster {
    pster, err := NewPster(dbpath)
    if err != nil { t.Fatal("Creating persister failed:", err) }
    return pster
}

func TestSimplePster(t *testing.T) {
    _ = initPster(t, "/tmp/testdb.gkv")
}
