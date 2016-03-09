package main

import (
    "github.com/critiqjo/cs733/assignment3/raft"
    "reflect"
    "testing"
)

func initPster(t *testing.T, dbpath string) *SimplePster {
    pster, err := NewPster(dbpath)
    if err != nil { t.Fatal("Creating persister failed:", err) }
    return pster
}

func TestSimplePster(t *testing.T) {
    dbpath := "/tmp/testdb.gkv"
    pster := initPster(t, dbpath)

    entry := raft.RaftEntry { Term: 0, CEntry: nil }
    ok := pster.LogUpdate(0, []raft.RaftEntry { entry })
    if !ok { t.Fatal("Failed to persist log entry") }

    pster_dup := initPster(t, dbpath)
    idx, entry_dup := pster_dup.LastEntry()
    if idx != 0 || !reflect.DeepEqual(entry_dup, &entry) {
        t.Fatal("Changes were not synced with disk!")
    }
    pster_dup.Close()

    entry.Term = 1
    entries := []raft.RaftEntry { entry, entry, entry }
    entries[1].CEntry = &raft.ClientEntry { UID: 1234, Data: "Yo!" }
    ok = pster.LogUpdate(1, entries)
    if !ok { t.Fatal("Failed to persist log entry") }

    pster_dup = initPster(t, dbpath)
    entries_dup, ok := pster_dup.LogSlice(1, 3)
    if !ok || !reflect.DeepEqual(entries_dup, entries) {
        t.Fatal("Changes were not synced with disk!")
    }
    pster_dup.Close()
}
