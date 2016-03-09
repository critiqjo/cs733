package main

import (
    "bufio"
    "bytes"
    "github.com/critiqjo/cs733/assignment3/raft"
    "reflect"
    "testing"
)

func init() {
    InitCoder()
}

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
            raft.RaftEntry { 1, &raft.ClientEntry { 1234, "some data" } },
            raft.RaftEntry { 1, &raft.ClientEntry { 2345, 1234567 } },
            raft.RaftEntry { 4, nil },
        }, 3,
    })
    testMsg(&raft.AppendReply { 1, true, 0, 1 })
    testMsg(&raft.VoteRequest { 7, 1, 8, 7 })
    testMsg(&raft.VoteReply { 8, false, 0 })
    testMsg(&raft.ClientEntry { 3456, nil })
}

func TestParseCEntry(t *testing.T) {
    buf := bytes.NewBuffer([]byte("write 0x543 345\r\nread 0x542\r\n"))
    rstream := bufio.NewReader(buf)
    centry, _ := ParseCEntry(rstream)
    if !reflect.DeepEqual(centry, &raft.ClientEntry { 0x543, "write 345" }) {
        t.Fatal("Bad write parsing!")
    }
    centry, _ = ParseCEntry(rstream)
    if !reflect.DeepEqual(centry, &raft.ClientEntry { 0x542, "read" }) {
        t.Fatal("Bad read parsing!")
    }
    centry, eof := ParseCEntry(rstream)
    if centry != nil || !eof {
        t.Fatal("Bad ending!")
    }
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
