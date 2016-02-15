package main

import "testing"
import "math/rand"
import "time"

type DummyMsger struct {
    notifch chan<- Message
    testch chan Message
}
func (self *DummyMsger) Register(notifch chan<- Message) {
    self.notifch = notifch
}
func (self *DummyMsger) SendAppendEntries(server int, msg *AppendEntries) {}
func (self *DummyMsger) BroadcastRequestVote(msg *RequestVote) {
    self.testch <- msg
}
func (self *DummyMsger) Client301(uid uint64, server int) {}
func (self *DummyMsger) Client503(uid uint64) {}

type DummyPster struct {}
func (pster *DummyPster) LogAppend(RaftLogEntry) {}
func (pster *DummyPster) LogDiscard(uint64) {}
func (pster *DummyPster) LogRead() []RaftLogEntry { return nil }
func (pster *DummyPster) StateRead() *PersistentState { return nil }
func (pster *DummyPster) StateSave(ps *PersistentState) {}

type DummyMachn struct {}
func (machn *DummyMachn) Apply(uid uint64, entry LogEntry) {}

func TestDummy(t *testing.T) {
    assert := func(e bool, args ...interface{}) {
        if !e {
            t.Fatal(args...)
        }
    }

    msger, pster, machn := &DummyMsger{ nil, make(chan Message) }, &DummyPster{}, &DummyMachn{}
    raft := NewRaftNode(0, msger, pster, machn)
    go raft.Run(func() time.Duration {
        return time.Duration(rand.Int() % 400 + 200) * time.Millisecond
    })

    m := <-msger.testch // wait for timeout
    assert(*m.(*RequestVote) == RequestVote { 1, 0, 0, 0 })
}
