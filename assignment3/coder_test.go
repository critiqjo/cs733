package main

import (
    "github.com/critiqjo/cs733/assignment3/raft"
    "reflect"
    "testing"
)

func TestCoding(t *testing.T) {
    // TODO
}

func TestParseCEntry(t *testing.T) {
    // TODO
}

func TestLogKeyCoding(t *testing.T) {
    blob := LogKeyEnc(2)
    if len(blob) != 8 {
        t.Fatal("Bad length of encoded key!")
    }
    if LogKeyDec(blob) != 2 {
        t.Fatal("Bad decoding of key!")
    }
}

func TestLogValCoding(t *testing.T) {
    testEntry := func(rentry *raft.RaftEntry) {
        blob, err := LogValEnc(rentry)
        if err != nil {
            t.Fatal(err)
        }
        rentry_dec, err := LogValDec(blob)
        if err != nil {
            t.Fatal(err)
        }
        if !reflect.DeepEqual(rentry, rentry_dec) {
            t.Fatal("Decoded into something else:", rentry_dec)
        }
    }
    testEntry(&raft.RaftEntry {
        Term: 123,
        CEntry: nil,
    })
    testEntry(&raft.RaftEntry {
        Term: 234,
        CEntry: &raft.ClientEntry {
            UID: 83459,
            Data: "this is my generic data!",
        },
    })
}
