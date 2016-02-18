package main

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

func (pster *DummyPster) LogAppend([]RaftEntry)   { }
func (pster *DummyPster) LogRead() []RaftEntry    { return nil }
func (pster *DummyPster) StateRead() *RaftState   { return nil }
func (pster *DummyPster) StateSave(ps *RaftState) { }
// }}}

type DummyMachn struct { // {{{
    msger *DummyMsger
}

func (self *DummyMachn) ApplyLazy(reqs []ClientEntry)  { self.msger.testch <- reqs }
func (self *DummyMachn) RespondIfSeen(uid uint64) bool { return false }
// }}}

func TestDummy(t *testing.T) {
    assert := func(e bool, args ...interface{}) {
        if !e {
            t.Fatal(args...)
        }
    }

    msger := &DummyMsger{ nil, make(chan interface{}) }
    pster, machn := &DummyPster{}, &DummyMachn{ msger }
    raft := NewRaftNode(0, msger, pster, machn)
    go raft.Run(func() time.Duration {
        return time.Duration(400) * time.Millisecond
    })

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
                Entry: &ClientEntry { 23425, nil },
            },
        },
        CommitIdx: 0,
    }
    m = <-msger.testch
    assert(*m.(*AppendReply) == AppendReply { 1, true }, "bad append", m)

    m = <-msger.testch // wait for timeout
    assert(*m.(*VoteRequest) == VoteRequest { 2, 0, 1, 1 }, "bad votereq", m)

    m = <-msger.testch // wait for timeout again
    assert(*m.(*VoteRequest) == VoteRequest { 3, 0, 1, 1 }, "bad votereq", m)

    msger.notifch <- &AppendEntries {
        Term: 3,
        LeaderId: 3,
        PrevLogIdx: 1,
        PrevLogTerm: 1,
        Entries: nil,
        CommitIdx: 1,
    }
    m = <-msger.testch
    assert(*m.(*AppendReply) == AppendReply { 3, true }, "bad append", m)
    m = <-msger.testch // apply lazy
    assert(m.([]ClientEntry)[0] == ClientEntry { 23425, nil }, "bad apply", m)
}
