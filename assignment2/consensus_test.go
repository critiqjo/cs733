package main

import "testing"

type DummyMsgr struct {
    notifch chan<- Message
}
func (msgr *DummyMsgr) Register(notifch chan<- Message) {
    msgr.notifch = notifch
}
func (msgr *DummyMsgr) SendAppendEntries(server int, message AppendEntries) {}
func (msgr *DummyMsgr) BroadcastRequestVote(message RequestVote) {}
func (msgr *DummyMsgr) RedirectTo(uid uint64, server int) {}

type DummyPstr struct {}
func (pstr *DummyPstr) LogAppend(RaftLogEntry) {}
func (pstr *DummyPstr) LogCompact(uint64) {}
func (pstr *DummyPstr) LogRead() []RaftLogEntry { return nil }
func (pstr *DummyPstr) StateRead() *PersistentState { return nil }
func (pstr *DummyPstr) StateSave(ps *PersistentState) {}

type DummyMachn struct {}
func (machn *DummyMachn) LogQueue(LogEntry) {}
func (machn *DummyMachn) LastApplied() *MachineState { return nil }

func TestDummy(t *testing.T) {
    msgr, pstr, machn := &DummyMsgr{ nil }, &DummyPstr{}, &DummyMachn{}
    raft := NewRaftNode(0, msgr, pstr, machn)
    raft.Run()
}
