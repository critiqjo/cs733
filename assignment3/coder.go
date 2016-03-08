package main

import (
    "bufio"
    "bytes"
    "encoding/binary"
    "encoding/gob"
    "encoding/json"
    "errors"
    "github.com/critiqjo/cs733/assignment3/raft"
    "regexp"
    "strconv"
)

func Encode(stuff interface{}) ([]byte, error) {
    // TODO replace json with gob: use gob.Register() for decoding into interface{}
    data, err := json.Marshal(stuff)
    if err != nil { return nil, err }
    switch stuff.(type) {
    case *raft.AppendEntries:
        return append([]byte("AE"), data...), nil
    case *raft.AppendReply:
        return append([]byte("AP"), data...), nil
    case *raft.ClientEntry:
        return append([]byte("CE"), data...), nil
    case *raft.VoteRequest:
        return append([]byte("VQ"), data...), nil
    case *raft.VoteReply:
        return append([]byte("VP"), data...), nil
    default:
        return nil, errors.New("Unknown stuff!")
    }
}

func Decode(data []byte) (raft.Message, error) {
    if len(data) < 2 { return nil, errors.New("Got nothin!") }
    hint := string(data[:2])
    switch hint {
    case "AE":
        var msg raft.AppendEntries
        err := json.Unmarshal(data[2:], &msg)
        return &msg, err
    case "AP":
        var msg raft.AppendReply
        err := json.Unmarshal(data[2:], &msg)
        return &msg, err
    case "CE":
        var msg raft.ClientEntry
        err := json.Unmarshal(data[2:], &msg)
        return &msg, err
    case "VQ":
        var msg raft.VoteRequest
        err := json.Unmarshal(data[2:], &msg)
        return &msg, err
    case "VP":
        var msg raft.VoteReply
        err := json.Unmarshal(data[2:], &msg)
        return &msg, err
    default:
        return nil, errors.New("Unintelligible data!")
    }
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
