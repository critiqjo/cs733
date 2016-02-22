package raft

import "testing"
import "time"
import "reflect"

// FIXME these tests are too dependent on the execution order; make it more general?

type DummyMsger struct { // {{{1
    notifch chan<- Message
    testch chan interface{}
}

func (self *DummyMsger) Register(notifch chan<- Message)       { self.notifch = notifch }
func (self *DummyMsger) Send(nodeId int, msg Message)          { self.testch <- msg } // `nodeId` is correct! :P
func (self *DummyMsger) BroadcastVoteRequest(msg *VoteRequest) { self.testch <- msg }
func (self *DummyMsger) Client301(uid uint64, nodeId int)      { } // this is correct too!!
func (self *DummyMsger) Client503(uid uint64)                  { } // you guessed it!!!

type DummyPster struct { } // {{{1

func (pster *DummyPster) LogUpdate([]RaftEntry)   { }
func (pster *DummyPster) LogRead() []RaftEntry    { return nil }
func (pster *DummyPster) StatusLoad() *RaftFields { return nil }
func (pster *DummyPster) StatusSave(RaftFields)   { }

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
    raft := NewNode(0, 5, 0, msger, pster, machn) // unbuffered channel
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
                Index: 1,
                Term: 1,
                Entry: &ClientEntry { 1234, nil },
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
            RaftEntry { 2, 3, nil }, // nothing to apply
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
            RaftEntry { 3, 3, nil },
        },
        CommitIdx: 2,
    }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 3, true, 0, 3 }, "Bad append 3t.3", m)
    assert(t, raft.log[3].Term == 3, "Bad log 3")

    msger.notifch <- &AppendEntries { // overwrite previous entry
        Term: 4,
        LeaderId: 3,
        PrevLogIdx: 2,
        PrevLogTerm: 3,
        Entries: []RaftEntry {
            RaftEntry { 3, 4, nil },
        },
        CommitIdx: 2,
    }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 4, true, 0, 3 }, "Bad append 4.1", m)
    assert(t, raft.log[3].Term == 4, "Bad log 4")
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
            RaftEntry { 1, 1, nil },
            RaftEntry { 2, 1, nil },
            RaftEntry { 3, 4, nil },
        },
        CommitIdx: 3,
    }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 4, true, 0, 3 }, "Bad append 4.2", m)

    m = <-msger.testch // wait for timeout
    assert_eq(t, m, &VoteRequest {
        Term: 5,
        CandidId: 0,
        LastLogIdx: 3,
        LastLogTerm: 4,
    }, "Bad votereq 5", m)

    m = <-msger.testch // wait for timeout again
    assert_eq(t, m, &VoteRequest { 6, 0, 3, 4 }, "Bad votereq 6", m)

    msger.notifch <- &AppendEntries {
        Term: 6,
        LeaderId: 3,
        PrevLogIdx: 3,
        PrevLogTerm: 4,
        Entries: nil,
        CommitIdx: 1,
    }
    m = <-msger.testch
    assert_eq(t, m, &AppendReply { 6, true, 0, 0 }, "Bad append 5", m)

    msger.notifch <- &VoteRequest { 6, 1, 3, 4 }
    m = <-msger.testch
    assert_eq(t, m, &VoteReply { 6, false, 0 }, "Bad votereply 6", m)

    m = <-msger.testch // wait for timeout one last time!
    assert_eq(t, m, &VoteRequest { 7, 0, 3, 4 }, "Bad votereq 6", m)

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
    assert(t, raft.state == Candidate, "Bad state 7.1", raft)
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

    msger.notifch <- &VoteReply { 1, true, 2 }
    m = <-msger.testch // wait for heartbeat
    assert(t, raft.state == Leader, "Bad state 1.2", raft)

    heartbeat := &AppendEntries { 1, 0, 0, 0, nil, 0 }
    assert_eq(t, m, heartbeat, "Bad heartbeat 1.1", m)
    assert_eq(t, <-msger.testch, heartbeat, "Bad heartbeat 1.2")
    assert_eq(t, <-msger.testch, heartbeat, "Bad heartbeat 1.3")
    assert_eq(t, <-msger.testch, heartbeat, "Bad heartbeat 1.4")

    clen := &ClientEntry { 1234, nil }
    apen := &AppendEntries { 1, 0, 0, 0, []RaftEntry {
        RaftEntry { 1, 1, clen },
    }, 0 }
    msger.notifch <- clen
    m = <-msger.testch
    assert_eq(t, m, apen, "Bad AppendEntries 1.1", m)
    assert_eq(t, <-msger.testch, apen, "Bad heartbeat 1.2")
    assert_eq(t, <-msger.testch, apen, "Bad heartbeat 1.3")
    assert_eq(t, <-msger.testch, apen, "Bad heartbeat 1.4")

    msger.notifch <- &AppendReply { 1, true, 1, 1 }
    msger.notifch <- &AppendReply { 1, true, 2, 1 }
    m = <-msger.testch // from ApplyLazy
    assert_eq(t, m, []ClientEntry { ClientEntry { 1234, nil } }, "Bad apply 1234", m)
    msger.notifch <- &AppendReply { 1, true, 3, 1 }
    msger.notifch <- &AppendReply { 1, true, 4, 1 }

    raft.Exit()
}
