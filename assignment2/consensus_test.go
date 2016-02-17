package main

import "testing"
import "math/rand"
import "time"

type DummyMsger struct {
    notifch chan<- Message
    testch chan interface{}
}
func (self *DummyMsger) Register(notifch chan<- Message) {
    self.notifch = notifch
}
func (self *DummyMsger) Send(server int, msg Message) {}
func (self *DummyMsger) BroadcastVoteRequest(msg *VoteRequest) {
    self.testch <- msg
}
func (self *DummyMsger) Client301(uid uint64, server int) {}
func (self *DummyMsger) Client503(uid uint64) {}

type DummyPster struct {}
func (pster *DummyPster) LogAppend([]RaftEntry) {}
func (pster *DummyPster) LogRead() []RaftEntry { return nil }
func (pster *DummyPster) StateRead() *RaftState { return nil }
func (pster *DummyPster) StateSave(ps *RaftState) {}

type DummyMachn struct {}
func (machn *DummyMachn) ApplyLazy(reqs []ClientEntry) {}
func (machn *DummyMachn) RespondIfSeen(uid uint64) bool { return false }

func TestDummy(t *testing.T) {
    assert := func(e bool, args ...interface{}) {
        if !e {
            t.Fatal(args...)
        }
    }

    msger, pster, machn := &DummyMsger{ nil, make(chan interface{}) }, &DummyPster{}, &DummyMachn{}
    raft := NewRaftNode(0, msger, pster, machn)
    go raft.Run(func() time.Duration {
        return time.Duration(rand.Int() % 400 + 200) * time.Millisecond
    })

    m := <-msger.testch // wait for timeout
    assert(*m.(*VoteRequest) == VoteRequest { 1, 0, 0, 0 })
}
