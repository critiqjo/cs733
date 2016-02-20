package raft

import "testing"
import "time"

type DummyMsger struct { // {{{
    notifch chan<- Message
    testch chan interface{}
}

func (self *DummyMsger) Register(notifch chan<- Message)       { self.notifch = notifch }
func (self *DummyMsger) Send(nodeId int, msg Message)          { self.testch <- msg } // `nodeId` is correct! :P
func (self *DummyMsger) BroadcastVoteRequest(msg *VoteRequest) { self.testch <- msg }
func (self *DummyMsger) Client301(uid uint64, nodeId int)      { } // this is correct too!!
func (self *DummyMsger) Client503(uid uint64)                  { } // you guessed it!!!
// }}}

type DummyPster struct { } // {{{

func (pster *DummyPster) LogUpdate([]RaftEntry)   { }
func (pster *DummyPster) LogRead() []RaftEntry    { return nil }
func (pster *DummyPster) StatusLoad() *RaftFields { return nil }
func (pster *DummyPster) StatusSave(RaftFields)   { }
// }}}

type DummyMachn struct { // {{{
    msger *DummyMsger
}

func (self *DummyMachn) ApplyLazy(entries []ClientEntry)  { self.msger.testch <- entries }
func (self *DummyMachn) RespondIfSeen(uid uint64) bool { return false }
// }}}

func assert(t *testing.T, e bool, args ...interface{}) {
    // Unidiomatic: https://golang.org/doc/faq#testing_framework
    if !e { t.Fatal(args...) }
}

func TestDummy(t *testing.T) {
    msger := &DummyMsger{ nil, make(chan interface{}) } // unbuffered channel: deliberate strictness
    pster, machn := &DummyPster{}, &DummyMachn{ msger }
    raft := NewRaftNode(0, 5, msger, pster, machn)
    go raft.Run(func(rs RaftState) time.Duration {
        return time.Duration(400) * time.Millisecond
    })

    followerTest(t, &raft, msger)
    candidateTest(t, &raft, msger)
}

func followerTest(t *testing.T, raft *RaftNode, msger *DummyMsger) {
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
    assert(t, *m.(*AppendReply) == AppendReply { 1, true, 0, 1 }, "Bad append 1", m)

    msger.notifch <- &AppendEntries {
        Term: 3,
        LeaderId: 3,
        PrevLogIdx: 1,
        PrevLogTerm: 1,
        Entries: []RaftEntry {
            RaftEntry {
                Index: 2,
                Term: 3,
                Entry: nil, // nothing to apply
            },
        },
        CommitIdx: 2, // commited till this entry
    }
    m = <-msger.testch
    assert(t, *m.(*AppendReply) == AppendReply { 3, true, 0, 2 }, "Bad append 3t.2", m)
    ces := (<-msger.testch).([]ClientEntry) // apply lazy of uid=1234
    assert(t, len(ces) == 1 && ces[0] == ClientEntry { 1234, nil }, "Bad apply 1234", ces)

    msger.notifch <- &AppendEntries { // heartbeat of old leader
        Term: 1,
        LeaderId: 2,
        PrevLogIdx: 1,
        PrevLogTerm: 1,
        Entries: nil,
        CommitIdx: 1,
    }
    m = <-msger.testch
    assert(t, *m.(*AppendReply) == AppendReply { 3, false, 0, 0 }, "Bad append 3f", m)

    msger.notifch <- &AppendEntries {
        Term: 3,
        LeaderId: 4,
        PrevLogIdx: 2,
        PrevLogTerm: 3,
        Entries: []RaftEntry {
            RaftEntry {
                Index: 3,
                Term: 3,
                Entry: nil,
            },
        },
        CommitIdx: 2,
    }
    m = <-msger.testch
    assert(t, *m.(*AppendReply) == AppendReply { 3, true, 0, 3 }, "Bad append 3t.3", m)
    assert(t, raft.log[3].Term == 3, "Bad log 3")

    msger.notifch <- &AppendEntries { // overwrite previous entry
        Term: 4,
        LeaderId: 3,
        PrevLogIdx: 2,
        PrevLogTerm: 3,
        Entries: []RaftEntry {
            RaftEntry {
                Index: 3,
                Term: 4,
                Entry: nil,
            },
        },
        CommitIdx: 2,
    }
    m = <-msger.testch
    assert(t, *m.(*AppendReply) == AppendReply { 4, true, 0, 3 }, "Bad append 4t", m)
    assert(t, raft.log[3].Term == 4, "Bad log 4")
}

func candidateTest(t *testing.T, raft *RaftNode, msger *DummyMsger) {
    var m interface{}
    m = <-msger.testch // wait for timeout
    assert(t, *m.(*VoteRequest) == VoteRequest { 5, 0, 3, 4 }, "Bad votereq 5", m)

    m = <-msger.testch // wait for timeout again
    assert(t, *m.(*VoteRequest) == VoteRequest { 6, 0, 3, 4 }, "Bad votereq 6", m)

    msger.notifch <- &AppendEntries {
        Term: 6,
        LeaderId: 3,
        PrevLogIdx: 3,
        PrevLogTerm: 4,
        Entries: nil,
        CommitIdx: 1,
    }
    m = <-msger.testch
    assert(t, *m.(*AppendReply) == AppendReply { 6, true, 0, 0 }, "Bad append 5", m)

    msger.notifch <- &VoteRequest {
        Term: 6,
        CandidId: 1,
        LastLogIdx: 3,
        LastLogTerm: 4,
    }
    m = <-msger.testch
    assert(t, *m.(*VoteReply) == VoteReply { 6, false, 0 }, "Bad votereply 6", m)

    m = <-msger.testch // wait for timeout one last time!
    assert(t, *m.(*VoteRequest) == VoteRequest { 7, 0, 3, 4 }, "Bad votereq 6", m)

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
    assert(t, raft.state == Candidate, "Bad state 7.1", m)

    msger.notifch <- &VoteReply { 7, true, 2 }
    msger.notifch <- &VoteReply { 7, true, 3 }
    msger.notifch <- &testEcho { }
    m = <-msger.testch // wait for echo
    assert(t, raft.state == Candidate, "Bad state 7.2", m)
    msger.notifch <- &VoteReply { 7, true, 4 }
    msger.notifch <- &testEcho { }
    m = <-msger.testch // wait for echo
    assert(t, raft.state == Leader, "Bad state 7.3", m)
    raft.Exit()
}
