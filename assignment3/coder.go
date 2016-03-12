package main

import (
    "bytes"
    "encoding/binary"
    "encoding/gob"
    "github.com/critiqjo/cs733/assignment3/raft"
    "regexp"
    "strconv"
)

func init() {
    gob.RegisterName("AE", new(raft.AppendEntries))
    gob.RegisterName("AP", new(raft.AppendReply))
    gob.RegisterName("CE", new(raft.ClientEntry))
    gob.RegisterName("VQ", new(raft.VoteRequest))
    gob.RegisterName("VP", new(raft.VoteReply))
    gob.RegisterName("MR", new(MachnRead))
    gob.RegisterName("MU", new(MachnUpdate))
}

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

// Tries to parse a ClientEntry from stream
func ParseCEntry(str string) *raft.ClientEntry {
    var pat *regexp.Regexp
    var matches []string
    pat = regexp.MustCompile("^read (0x[0-9a-f]+)$")
    matches = pat.FindStringSubmatch(str)
    if len(matches) > 0 {
        uid, _ := strconv.ParseUint(matches[1], 0, 64)
        return &raft.ClientEntry {
            UID: uid,
            Data: &MachnRead {},
        }
    }

    pat = regexp.MustCompile("^update (0x[0-9a-f]+) ([0-9]+)$")
    matches = pat.FindStringSubmatch(str)
    if len(matches) > 0 {
        uid, _ := strconv.ParseUint(matches[1], 0, 64)
        val, _ := strconv.ParseInt(matches[2], 0, 64)
        return &raft.ClientEntry {
            UID: uid,
            Data: &MachnUpdate { val },
        }
    }

    return nil
}

func binaryMustEnc(val interface{}, initCap int) []byte {
    buf := bytes.NewBuffer(make([]byte, 0, initCap))
    err := binary.Write(buf, binary.BigEndian, val)
    if err != nil { panic("Impossible encode error!") }
    return buf.Bytes()
}

func binaryMustDec(blob []byte, val interface{}) {
    buf := bytes.NewBuffer(blob)
    err := binary.Read(buf, binary.BigEndian, val)
    if err != nil { panic("Impossible decode error!") }
}

func U64Enc(val uint64) []byte {
    return binaryMustEnc(val, 8)
}

func U64Dec(blob []byte) uint64 {
    val := new(uint64)
    binaryMustDec(blob, val)
    return *val
}

func LogValEnc(entry *raft.RaftEntry) ([]byte, error) {
    buf := new(bytes.Buffer)
    enc := gob.NewEncoder(buf)
    err := enc.Encode(entry)
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

func FieldsEnc(fields *raft.RaftFields) []byte {
    return binaryMustEnc(fields, 12)
}

func FieldsDec(blob []byte) *raft.RaftFields {
    fields := new(raft.RaftFields)
    binaryMustDec(blob, fields)
    return fields
}
