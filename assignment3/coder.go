package main

import (
    "bufio"
    "github.com/critiqjo/cs733/assignment3/raft"
    "regexp"
    "strconv"
)

func Encode(stuff interface{}) []byte {
    return []byte("Yo!")
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
