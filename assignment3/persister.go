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
    if n == 0 {
        item, _ := self.rlog.GetItem(LogKeyEnc(startIdx - 1), false)
        if item != nil {
            return nil, true
        } else {
            return nil, false
        }
    } else {
        var entries []raft.RaftEntry
        var idx = startIdx
        iter_cb := func(item *gkvlite.Item) bool {
            if idx != LogKeyDec(item.Key) { panic("Corrupt log!") }
            entry, err := LogValDec(item.Val)
            if err != nil { panic("Corrupt log entry!") }
            entries = append(entries, *entry)
            idx += 1
            return idx - startIdx < uint64(n)
        }
        self.rlog.VisitItemsAscend(LogKeyEnc(startIdx), true, iter_cb)
        return entries, true
    }
}

func (self *SimplePster) LogUpdate(startIdx uint64, slice []raft.RaftEntry) bool {
    tailItem, _ := self.rlog.MaxItem(false)
    var tailIdx int64 = -1 // physically not possible to overflow int64 anyway
    if tailItem != nil {
        tailIdx = int64(LogKeyDec(tailItem.Key))
    }

    if tailIdx + 1 >= int64(startIdx) {
        if len(slice) == 0 {
            return true // nothing to update
        }
        if tailIdx >= 0 { // truncate
            newTailIdx := startIdx + uint64(len(slice)) - 1
            for idx := uint64(tailIdx); idx > newTailIdx; idx -= 1 {
                deleted, _ := self.rlog.Delete(LogKeyEnc(idx))
                if !deleted { panic("Corrupt log!") }
            }
        }
        idx := startIdx
        for _, entry := range slice { // append/update
            blob, err := LogValEnc(&entry)
            if err != nil { panic("Impossible encode error!!") }
            err = self.rlog.Set(LogKeyEnc(idx), blob)
            if err != nil { return false } // panic??
            idx += 1
        }
        err := self.store.Flush()
        return err == nil
    }
    return false
}

func (self *SimplePster) GetFields() *raft.RaftFields {
    blob, _ := self.rfields.Get([]byte {0})
    if blob == nil { return nil }
    return FieldsDec(blob)
}

func (self *SimplePster) SetFields(fields raft.RaftFields) bool {
    err := self.rfields.Set([]byte {0}, FieldsEnc(&fields))
    if err != nil { return false }
    err = self.store.Flush()
    return err == nil
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

func (self *SimplePster) Close() { // {{{1
    self.store.Close()
}
