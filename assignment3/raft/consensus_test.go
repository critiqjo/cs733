package raft

import "testing"
import "time"
import "reflect"

// FIXME these tests are too dependent on the execution order
// TODO test with even number of nodes

type DummyMsger struct { // {{{1
    notifch chan<- Message
    testch chan interface{}
}

func (self *DummyMsger) Register(notifch chan<- Message)       { self.notifch = notifch }
func (self *DummyMsger) Send(nodeId int, msg Message)          { self.testch <- msg } // `nodeId` is correct! :P
func (self *DummyMsger) BroadcastVoteRequest(msg *VoteRequest) { self.testch <- msg }
func (self *DummyMsger) Client301(uid uint64, nodeId int)      { } // this is correct too!!
func (self *DummyMsger) Client503(uid uint64)                  { } // you guessed it!!!

type DummyPster struct { // {{{1
    log []RaftEntry
}

func (self *DummyPster) Entry(idx uint64) *RaftEntry {
    return &self.log[idx]
}
func (self *DummyPster) LastEntry() (uint64, *RaftEntry) {
    if len(self.log) == 0 { return 0, nil }
    lastIdx := len(self.log) - 1
    return uint64(lastIdx), &self.log[lastIdx]
}
func (self *DummyPster) LogSlice(startIdx uint64, n int) ([]RaftEntry, bool) {
    if n == 0 {
        if int(startIdx) < len(self.log) + 1 {
            return nil, true
        }
    } else if n > 0 {
        if int(startIdx) < len(self.log) {
            endIdx := int(startIdx) + n
            if endIdx > len(self.log) {
                endIdx = len(self.log)
            }
            return self.log[startIdx:endIdx], true
        }
    }
    return nil, false
}
func (self *DummyPster) LogUpdate(startIdx uint64, slice []RaftEntry) bool {
    if startIdx == 0 {
        self.log = slice
    } else {
        self.log = append(self.log[0:int(startIdx)], slice...)
    }
    return true
}
func (self *DummyPster) GetFields() *RaftFields { return nil }
func (self *DummyPster) SetFields(RaftFields) bool { return true }

type DummyMachn struct { // {{{1
    msger *DummyMsger
}

func (self *DummyMachn) ApplyLazy(entries []ClientEntry)  { self.msger.testch <- entries }
func (self *DummyMachn) RespondIfSeen(uid uint64) bool { return false }

// ---- utility functions {{{1
func assert(t *testing.T, e bool, args ...interface{}) {
    // Unidiomatic: https://golang.org/doc/faq#testing_framework
    if !e { t.Fatal(args...) }
}

func assert_eq(t *testing.T, x, y interface{}, args ...interface{}) {
    assert(t, reflect.DeepEqual(x, y), args...)
}

func initTest() (*RaftNode, *DummyMsger, *DummyPster, *DummyMachn) {
    // Note: Deadlocking due to unbuffered channels is considered a bug!
    msger := &DummyMsger{ nil, make(chan interface{}) } // unbuffered channel
    pster, machn := &DummyPster{}, &DummyMachn{ msger }
    raft, err := NewNode(0, []int { 0, 1, 2, 3, 4 }, 0, msger, pster, machn) // unbuffered channel
    if err != nil { panic(err) }
    go raft.Run(func(rs RaftState) time.Duration {
        return time.Duration(400) * time.Millisecond
    })
    return raft, msger, pster, machn
}

func TestFollower(t *testing.T) { // {{{1
    raft, msger, _, _ := initTest()
    var m interface{}

    msger.notifch <- &AppendEntries {
        Term: 1,
        LeaderId: 2,
        PrevLogIdx: 0,
        PrevLogTerm: 0,
        Entries: []RaftEntry {
            RaftEntry {
                Term: 1,
                CEntry: &ClientEntry { 1234, nil },
            },
        },
        CommitIdx: 0,
    }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 1, true, 0, 1 }, "Bad append 1", m)

    msger.notifch <- &AppendEntries {
        Term: 3,
        LeaderId: 3,
        PrevLogIdx: 1,
        PrevLogTerm: 1,
        Entries: []RaftEntry {
            RaftEntry { 3, nil }, // nothing to apply
        },
        CommitIdx: 2, // commited till this entry
    }
    // Note: if the calling order of Send(.., &AppendReply) and applyCommitted(..)
    //       changes in followerHandler, this will fail.
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 3, true, 0, 2 }, "Bad append 3t.2", m)
    m = <-msger.testch // from ApplyLazy
    assert_eq(t, m, []ClientEntry { ClientEntry { 1234, nil } }, "Bad apply 1234", m)

    msger.notifch <- &AppendEntries { // heartbeat of old leader
        Term: 1,
        LeaderId: 2,
        PrevLogIdx: 1,
        PrevLogTerm: 1,
        Entries: nil,
        CommitIdx: 1,
    }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 3, false, 0, 0 }, "Bad append 3f", m)

    msger.notifch <- &AppendEntries {
        Term: 3,
        LeaderId: 4,
        PrevLogIdx: 2,
        PrevLogTerm: 3,
        Entries: []RaftEntry {
            RaftEntry { 3, nil },
        },
        CommitIdx: 2,
    }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 3, true, 0, 3 }, "Bad append 3t.3", m)
    assert(t, raft.log(3).Term == 3, "Bad log 3")

    msger.notifch <- &AppendEntries { // overwrite previous entry
        Term: 4,
        LeaderId: 3,
        PrevLogIdx: 2,
        PrevLogTerm: 3,
        Entries: []RaftEntry {
            RaftEntry { 4, nil }, // 3
        },
        CommitIdx: 2,
    }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 4, true, 0, 3 }, "Bad append 4.1", m)
    assert(t, raft.log(3).Term == 4, "Bad log 4")

    msger.notifch <- &AppendEntries { // a lot happened!!
        Term: 8,
        LeaderId: 2,
        PrevLogIdx: 11,
        PrevLogTerm: 8,
        Entries: nil,
        CommitIdx: 10,
    }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 8, false, 0, 0 }, "Bad append 8.1", m)

    msger.notifch <- &AppendEntries {
        Term: 8,
        LeaderId: 2,
        PrevLogIdx: 3,
        PrevLogTerm: 4,
        Entries: []RaftEntry {
            RaftEntry { 4, &ClientEntry { 1235, nil } }, // 4
            RaftEntry { 4, &ClientEntry { 1236, nil } }, // 5
            RaftEntry { 6, &ClientEntry { 1237, nil } }, // 6
            RaftEntry { 6, &ClientEntry { 1238, nil } }, // 7
        },
        CommitIdx: 10,
    }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 8, true, 0, 7 }, "Bad append 8.2", m)
    m = <-msger.testch // from ApplyLazy
    assert(t, len(m.([]ClientEntry)) == 4, "Bad apply 123*", m)
    assert(t, raft.votedFor == 2, "Bad votedFor 8.2", raft)

    msger.notifch <- &VoteRequest { 7, 1, 8, 7 } // stale term
    m = <-msger.testch
    assert_eq(t, m, &VoteReply { 8, false, 0 }, "Bad votereply 8.1", m)

    msger.notifch <- &VoteRequest { 8, 1, 7, 6 }
    m = <-msger.testch
    assert_eq(t, m, &VoteReply { 8, false, 0 }, "Bad votereply 8.2", m)

    msger.notifch <- &VoteRequest { 9, 1, 6, 6 } // not up to date
    m = <-msger.testch
    assert_eq(t, m, &VoteReply { 9, false, 0 }, "Bad votereply 9.1", m)

    msger.notifch <- &VoteRequest { 9, 3, 7, 6 }
    m = <-msger.testch
    assert_eq(t, m, &VoteReply { 9, true, 0 }, "Bad votereply 9.2", m)
    assert(t, raft.votedFor == 3, "Bad votedFor 9.3", raft)

    msger.notifch <- &VoteRequest { 9, 4, 7, 6 } // already voted
    m = <-msger.testch
    assert_eq(t, m, &VoteReply { 9, false, 0 }, "Bad votereply 9.3", m)

    raft.Exit()
}

func TestCandidate(t *testing.T) { // {{{1
    raft, msger, _, _ := initTest()
    var m interface{}

    msger.notifch <- &AppendEntries {
        Term: 4,
        LeaderId: 2,
        PrevLogIdx: 0,
        PrevLogTerm: 0,
        Entries: []RaftEntry {
            RaftEntry { 1, nil }, // 1
            RaftEntry { 1, nil }, // 2
            RaftEntry { 4, nil }, // 3
        },
        CommitIdx: 3,
    }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 4, true, 0, 3 }, "Bad append 4", m)
    assert(t, raft.state == Follower, "Bad state 4", raft)

    m = <-msger.testch // wait for timeout
    assert_eq(t, m, &VoteRequest {
        Term: 5,
        CandidId: 0,
        LastLogIdx: 3,
        LastLogTerm: 4,
    }, "Bad votereq 5", m)
    assert(t, raft.state == Candidate, "Bad state 5", raft)

    msger.notifch <- &AppendEntries { 4, 2, 3, 4, nil, 3 }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 5, false, 0, 0 }, "Bad append 5", m)

    m = <-msger.testch // wait for timeout again
    assert_eq(t, m, &VoteRequest { 6, 0, 3, 4 }, "Bad votereq 6", m)

    msger.notifch <- &AppendEntries { 6, 3, 3, 4, nil, 1 }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 6, true, 0, 0 }, "Bad append 6", m)
    assert(t, raft.state == Follower, "Bad state 6", raft)

    m = <-msger.testch // wait for timeout one last time!
    assert_eq(t, m, &VoteRequest { 7, 0, 3, 4 }, "Bad votereq 7", m)

    msger.notifch <- &VoteRequest { 7, 1, 3, 4 }
    m = <-msger.testch
    assert_eq(t, m, &VoteReply { 7, false, 0 }, "Bad votereply 7", m)

    msger.notifch <- &VoteReply {
        Term: 6, // old term
        Granted: true,
        NodeId: 1,
    }
    msger.notifch <- &VoteReply { 6, true, 2 }
    msger.notifch <- &VoteReply { 6, true, 3 }
    msger.notifch <- &VoteReply { 6, true, 4 }
    msger.notifch <- &testEcho { }
    m = <-msger.testch // wait for echo
    assert(t, raft.state == Candidate, "Bad state 7", raft)

    msger.notifch <- &VoteRequest { 8, 1, 3, 4 }
    m = <-msger.testch
    assert_eq(t, m, &VoteReply { 8, true, 0 }, "Bad votereply 7", m)
    assert(t, raft.state == Follower, "Bad state 8", raft)

    raft.Exit()
}

func TestLeader(t *testing.T) { // {{{1
    raft, msger, _, _ := initTest()
    var m interface{}

    m = <-msger.testch // wait for timeout
    assert_eq(t, m, &VoteRequest { 1, 0, 0, 0 }, "Bad votereq 1", m)

    msger.notifch <- &VoteReply { 1, true, 1 }
    msger.notifch <- &testEcho { }
    m = <-msger.testch
    assert(t, raft.state == Candidate, "Bad state 1.1", raft)

    msger.notifch <- &VoteReply { 1, true, 2 } // gets majority; broadcasts heartbeats
    hb := &AppendEntries { 1, 0, 0, 0, nil, 0 } // term, id, prevIdx, prevTerm, entries, commitIdx
    assert_eq(t, <-msger.testch, hb, "Bad heartbeat 1.1")
    assert_eq(t, <-msger.testch, hb, "Bad heartbeat 1.2")
    assert_eq(t, <-msger.testch, hb, "Bad heartbeat 1.3")
    assert_eq(t, <-msger.testch, hb, "Bad heartbeat 1.4")
    assert(t, raft.state == Leader, "Bad state 1.2", raft)

    clen := &ClientEntry { 1234, nil }
    msger.notifch <- clen
    apen := &AppendEntries { 1, 0, 0, 0, []RaftEntry { RaftEntry { 1, clen } }, 0 }
    assert_eq(t, <-msger.testch, apen, "Bad AppendEntries 1.1")
    assert_eq(t, <-msger.testch, apen, "Bad AppendEntries 1.2")
    assert_eq(t, <-msger.testch, apen, "Bad AppendEntries 1.3")
    assert_eq(t, <-msger.testch, apen, "Bad AppendEntries 1.4")

    msger.notifch <- clen // duplicate; should ignore
    msger.notifch <- &testEcho { }
    m = <-msger.testch // wait for echo

    msger.notifch <- &AppendReply { 1, true, 1, 1 }
    msger.notifch <- &AppendReply { 1, true, 2, 1 }
    m = <-msger.testch // majority reached, ApplyLazy
    assert_eq(t, m, []ClientEntry { *clen }, "Bad apply 1234", m)

    clen = &ClientEntry { 1235, nil }
    msger.notifch <- &AppendEntries { 3, 1, 1, 1,
        []RaftEntry {
            RaftEntry { 2, nil }, // 2
            RaftEntry { 3, nil }, // 3
            RaftEntry { 3, nil }, // 4
            RaftEntry { 3, clen }, // 5
        }, 4,
    }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 3, true, 0, 5 }, "Bad append 3", m)
    assert(t, raft.state == Follower, "Bad state 3", raft)

    m = <-msger.testch // wait for timeout
    assert_eq(t, m, &VoteRequest { 4, 0, 5, 3 }, "Bad votereq 1", m)

    msger.notifch <- &VoteReply { 4, true, 1 }
    msger.notifch <- &VoteReply { 4, true, 2 } // gets majority
    hb = &AppendEntries { 4, 0, 5, 3, nil, 4 }
    assert_eq(t, <-msger.testch, hb, "Bad heartbeat 4.1")
    assert_eq(t, <-msger.testch, hb, "Bad heartbeat 4.2")
    assert_eq(t, <-msger.testch, hb, "Bad heartbeat 4.3")
    assert_eq(t, <-msger.testch, hb, "Bad heartbeat 4.4")

    msger.notifch <- clen // duplicate; should ignore
    msger.notifch <- &testEcho { }
    m = <-msger.testch // wait for echo

    msger.notifch <- &AppendReply { 4, false, 1, 0 }
    assert_eq(t, <-msger.testch, &AppendEntries { 4, 0, 4, 3, nil, 4 }, "Bad append 4.1")
    msger.notifch <- &AppendReply { 4, false, 1, 0 }
    assert_eq(t, <-msger.testch, &AppendEntries { 4, 0, 3, 3, nil, 4 }, "Bad append 4.2")
    msger.notifch <- &AppendReply { 4, false, 1, 0 }
    assert_eq(t, <-msger.testch, &AppendEntries { 4, 0, 2, 2, nil, 4 }, "Bad append 4.3")
    msger.notifch <- &AppendReply { 4, true, 1, 0 }
    assert_eq(t, <-msger.testch, &AppendEntries {
        4, 0, 2, 2,
        []RaftEntry {
            RaftEntry { 3, nil }, // 3
            RaftEntry { 3, nil }, // 4
            RaftEntry { 3, clen }, // 5
        }, 4,
    }, "Bad append 4.4")

    msger.notifch <- &AppendReply { 5, false, 2, 0 }
    msger.notifch <- &testEcho { }
    m = <-msger.testch // wait for echo
    assert(t, raft.term == 5, "Bad term 5", raft)
    assert(t, raft.state == Follower, "Bad state 5")

    raft.Exit()
}