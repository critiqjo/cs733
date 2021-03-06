package main

import (
    "github.com/critiqjo/cs733/assignment3/raft"
    "reflect"
    "testing"
)

func TestCoding(t *testing.T) {
    testMsg := func(msg raft.Message) {
        blob, err := MsgEnc(msg)
        if err != nil { t.Fatal(err) }
        msg_dec, err := MsgDec(blob)
        if err != nil { t.Fatal(err) }
        if !reflect.DeepEqual(msg_dec, msg) {
            t.Fatal("Bad decoding of msg!")
        }
    }
    testMsg(&raft.AppendEntries {
        4, 2, 0, 0, []raft.RaftEntry {
            raft.RaftEntry { 1, &raft.ClientEntry { 1234, &MachnRead {} } },
            raft.RaftEntry { 1, &raft.ClientEntry { 2345, &MachnUpdate { -2 } } },
            raft.RaftEntry { 4, nil },
        }, 3,
    })
    testMsg(&raft.AppendReply { 1, true, 0, 1 })
    testMsg(&raft.VoteRequest { 7, 1, 8, 7 })
    testMsg(&raft.VoteReply { 8, false, 0 })
    testMsg(&raft.ClientEntry { 3456, nil })
}

func TestParseCEntry(t *testing.T) {
    centry := ParseCEntry("update 0x543 345")
    if !reflect.DeepEqual(centry, &raft.ClientEntry { 0x543, &MachnUpdate { 345 } }) {
        t.Fatal("Bad write parsing!")
    }
    centry = ParseCEntry("read 0x542")
    if !reflect.DeepEqual(centry, &raft.ClientEntry { 0x542, &MachnRead {} }) {
        t.Fatal("Bad read parsing!")
    }
}

func TestU64Coding(t *testing.T) {
    blob := U64Enc(7)
    if len(blob) != 8 {
        t.Fatal("Bad length of encoded key!")
    }
    if U64Dec(blob) != 7 {
        t.Fatal("Bad decoding of key!")
    }
}

func TestLogValCoding(t *testing.T) {
    testEntry := func(entry *raft.RaftEntry) {
        blob, err := LogValEnc(entry)
        if err != nil {
            t.Fatal(err)
        }
        entry_dec, err := LogValDec(blob)
        if err != nil {
            t.Fatal(err)
        }
        if !reflect.DeepEqual(entry, entry_dec) {
            t.Fatal("Decoded into something else:", entry_dec)
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
