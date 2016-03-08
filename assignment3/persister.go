package main

import (
    "github.com/critiqjo/cs733/assignment3/raft"
    "github.com/steveyen/gkvlite"
    "os"
)

type SimplePster struct {
    file    *os.File
    store   *gkvlite.Store
    rlog    *gkvlite.Collection
    rfields *gkvlite.Collection
}

// ---- quack like a Persister {{{1
func (self *SimplePster) Entry(idx uint64) *raft.RaftEntry {
    blob, _ := self.rlog.Get(LogKeyEnc(idx))
    if blob == nil { return nil }
    entry, err := LogValDec(blob)
    if err != nil {
        // panic?
        return nil
    }
    return entry
}

func (self *SimplePster) LastEntry() (uint64, *raft.RaftEntry) {
    item, _ := self.rlog.MaxItem(true)
    if item == nil {
        return 0, nil
    }
    idx := LogKeyDec(item.Key)
    entry, err := LogValDec(item.Val)
    if err != nil {
        // panic?
        return 0, nil
    }
    return idx, entry
}

func (self *SimplePster) LogSlice(startIdx uint64, n int) ([]raft.RaftEntry, bool) {
    return nil, false
}

func (self *SimplePster) LogUpdate(startIdx uint64, slice []raft.RaftEntry) bool {
    return false
}

func (self *SimplePster) GetFields() *raft.RaftFields {
    return nil
}

func (self *SimplePster) SetFields(raft.RaftFields) bool {
    return false
}

func NewPster(dbpath string) (*SimplePster, error) { // {{{1
    var store *gkvlite.Store
    file, err := os.OpenFile(dbpath, os.O_RDWR|os.O_CREATE|os.O_SYNC, 0660)
    if err != nil {
        return nil, err
    }
    store, err = gkvlite.NewStore(file)
    if err != nil {
        return nil, err
    }
    return &SimplePster {
        file: file,
        store: store,
        rlog: store.SetCollection("rlog", nil),
        rfields: store.SetCollection("rfields", nil),
    }, nil
}
