package raft

import "testing"
import "time"

type DummyMsger struct { // {{{
    notifch chan<- Message
    testch chan interface{}
}

func (self *DummyMsger) Register(notifch chan<- Message)       { self.notifch = notifch }
func (self *DummyMsger) Send(server int, msg Message)          { self.testch <- msg }
func (self *DummyMsger) BroadcastVoteRequest(msg *VoteRequest) { self.testch <- msg }
func (self *DummyMsger) Client301(uid uint64, server int)      { }
func (self *DummyMsger) Client503(uid uint64)                  { }
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
    msger := &DummyMsger{ nil, make(chan interface{}) }
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
    assert(t, *m.(*AppendReply) == AppendReply { 1, true }, "Bad append 1", m)
    // - - - - - - - - - - - - Ha ha ha ha - - - - - - - - - - - - - - ^^^ - !!

    msger.notifch <- &AppendEntries { // heartbeat
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
    assert(t, *m.(*AppendReply) == AppendReply { 3, true }, "Bad append 3", m)
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
    assert(t, *m.(*AppendReply) == AppendReply { 3, false }, "Bad append 3f", m)
}

func candidateTest(t *testing.T, raft *RaftNode, msger *DummyMsger) {
    var m interface{}
    m = <-msger.testch // wait for timeout
    assert(t, *m.(*VoteRequest) == VoteRequest { 4, 0, 2, 3 }, "Bad votereq 4", m)

    m = <-msger.testch // wait for timeout again
    assert(t, *m.(*VoteRequest) == VoteRequest { 5, 0, 2, 3 }, "Bad votereq 5", m)

    msger.notifch <- &AppendEntries {
        Term: 5,
        LeaderId: 3,
        PrevLogIdx: 1,
        PrevLogTerm: 1,
        Entries: nil,
        CommitIdx: 1,
    }
    m = <-msger.testch
    assert(t, *m.(*AppendReply) == AppendReply { 5, true }, "Bad append 5", m)
}
