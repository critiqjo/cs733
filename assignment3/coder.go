package main

import (
    "bufio"
    "bytes"
    "encoding/binary"
    "encoding/gob"
    "github.com/critiqjo/cs733/assignment3/raft"
    "regexp"
    "strconv"
)

type happyWrap struct { // make gob happy! Is there an easier way?
    Smile interface{}
}

func MsgEnc(msg raft.Message) ([]byte, error) {
    buf := new(bytes.Buffer)
    enc := gob.NewEncoder(buf)
    err := enc.Encode(&happyWrap { msg })
    if err != nil { return nil, err }
    return buf.Bytes(), nil
}

func MsgDec(blob []byte) (raft.Message, error) {
    var happy = new(happyWrap)
    dec := gob.NewDecoder(bytes.NewBuffer(blob))
    err := dec.Decode(happy)
    if err != nil { return nil, err }
    return happy.Smile, nil
}

// This should be called once to Register types with gob
func InitCoder() {
    gob.RegisterName("AE", new(raft.AppendEntries))
    gob.RegisterName("AP", new(raft.AppendReply))
    gob.RegisterName("CE", new(raft.ClientEntry))
    gob.RegisterName("VQ", new(raft.VoteRequest))
    gob.RegisterName("VP", new(raft.VoteReply))
    // Register type of ClientEntry.Data if it is not a built-in
}

// Tries to parse a ClientEntry from stream
// if connection is dead, then returns (nil, true)
// else if parsing succeeds, then returns (<parsed entry>, false)
// else returns (nil, false)
func ParseCEntry(rstream *bufio.Reader) (*raft.ClientEntry, bool) {
    str, err := rstream.ReadString('\n')
    if err != nil { return nil, true }

    var pat *regexp.Regexp
    var matches []string
    pat = regexp.MustCompile("^read (0x[0-9a-f]+)\r\n$")
    matches = pat.FindStringSubmatch(str)
    if len(matches) > 0 {
        uid, _ := strconv.ParseUint(matches[1], 0, 64)
        return &raft.ClientEntry {
            UID: uid,
            Data: "read",
        }, false
    }

    pat = regexp.MustCompile("^write (0x[0-9a-f]+) ([0-9]+)\r\n$")
    matches = pat.FindStringSubmatch(str)
    if len(matches) > 0 {
        uid, _ := strconv.ParseUint(matches[1], 0, 64)
        return &raft.ClientEntry {
            UID: uid,
            Data: "write " + matches[2],
        }, false
    }

    return nil, false
}

func LogKeyEnc(val uint64) []byte {
    buf := bytes.NewBuffer(make([]byte, 0, 8))
    err := binary.Write(buf, binary.BigEndian, val)
    if err != nil { panic("Impossible encode error!") }
    return buf.Bytes()
}

func LogKeyDec(blob []byte) uint64 {
    buf := bytes.NewBuffer(blob)
    val := new(uint64)
    err := binary.Read(buf, binary.BigEndian, val)
    if err != nil { panic("(Not so) impossible decode error!") }
    return *val
}

func LogValEnc(rentry *raft.RaftEntry) ([]byte, error) {
    buf := new(bytes.Buffer)
    enc := gob.NewEncoder(buf)
    err := enc.Encode(rentry)
    if err != nil { return nil, err }
    return buf.Bytes(), nil
}

func LogValDec(blob []byte) (*raft.RaftEntry, error) {
    re := new(raft.RaftEntry)
    dec := gob.NewDecoder(bytes.NewBuffer(blob))
    err := dec.Decode(re)
    if err != nil { return nil, err }
    return re, nil
}
