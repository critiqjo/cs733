package main

import (
    "bufio"
    "encoding/json"
    "errors"
    "github.com/critiqjo/cs733/assignment3/raft"
    "regexp"
    "strconv"
)

func Encode(stuff interface{}) ([]byte, error) {
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
