package main

import (
    "github.com/critiqjo/cs733/assignment3/raft"
    "reflect"
    "testing"
)

// ---- utility functions {{{1
func assert(t *testing.T, e bool, args ...interface{}) {
    // Unidiomatic: https://golang.org/doc/faq#testing_framework
    if !e { t.Fatal(args...) }
}

func assert_eq(t *testing.T, x, y interface{}, args ...interface{}) {
    assert(t, reflect.DeepEqual(x, y), args...)
}

func initTest(t *testing.T, cluster map[int]Node, nodeId int) (*SimpleMsger, chan raft.Message) {
    raftch := make(chan raft.Message)
    msger, err := NewMsger(nodeId, cluster)
    if err != nil { t.Fatal("Creating messenger failed:", err) }
    msger.Register(raftch)
    msger.SpawnListeners()
    return msger, raftch
}

func TestSimple(t *testing.T) { // {{{1
    cluster := map[int]Node {
        1: Node { Host: "127.0.0.1", RPort: 1234, CPort: 1235 },
        2: Node { Host: "127.0.0.1", RPort: 2345, CPort: 2346 },
        3: Node { Host: "127.0.0.1", RPort: 3456, CPort: 3457 },
    }

    msger1, raftch1 := initTest(t, cluster, 1)
    msger2, raftch2 := initTest(t, cluster, 2)
    _, raftch3 := initTest(t, cluster, 3)

    apen := &raft.AppendEntries {
        4, 2, 0, 0, []raft.RaftEntry {
            raft.RaftEntry { 1, nil },
            raft.RaftEntry { 1, nil },
            raft.RaftEntry { 4, nil },
        }, 3,
    }
    msger1.Send(2, apen)
    m := <-raftch2
    assert_eq(t, m, apen, "Message mismatch", m)

    vreq := &raft.VoteRequest { 7, 1, 8, 7 }
    msger2.BroadcastVoteRequest(vreq)
    m = <-raftch1
    assert_eq(t, m, vreq, "VoteReq mismatch", m)
    m = <-raftch3
    assert_eq(t, m, vreq, "VoteReq mismatch", m)
}
