package main

import (
	"github.com/critiqjo/cs733/assignment4/raft"
	"github.com/steveyen/gkvlite"
	"log"
	"os"
)

const NilIdx = ^uint64(0)

type SimplePster struct {
	file    *os.File
	store   *gkvlite.Store
	rlog    *gkvlite.Collection
	rfields *gkvlite.Collection
	err     *log.Logger
}

func (self *SimplePster) lastIdx() uint64 { // {{{1
	tailItem, _ := self.rlog.MaxItem(false)
	var tailIdx uint64 = NilIdx
	if tailItem != nil {
		tailIdx = uint64(U64Dec(tailItem.Key))
	}
	return tailIdx
}

// ---- quack like a Persister {{{1
func (self *SimplePster) Entry(idx uint64) *raft.RaftEntry {
	blob, _ := self.rlog.Get(U64Enc(idx))
	if blob == nil {
		return nil
	}
	entry, err := LogValDec(blob)
	if err != nil {
		self.err.Print(err.Error())
		return nil // panic?
	}
	return entry
}

func (self *SimplePster) LastEntry() (uint64, *raft.RaftEntry) {
	item, _ := self.rlog.MaxItem(true)
	if item == nil {
		return 0, nil
	}
	idx := U64Dec(item.Key)
	entry, err := LogValDec(item.Val)
	if err != nil {
		self.err.Print(err.Error())
		return 0, nil // panic?
	}
	return idx, entry
}

func (self *SimplePster) LogSlice(startIdx uint64, endIdx uint64) ([]raft.RaftEntry, bool) {
	lastIdx := self.lastIdx()
	if lastIdx == NilIdx {
		if startIdx == 0 && endIdx == 0 {
			return nil, true
		} else {
			return nil, false
		}
	} else if startIdx > endIdx {
		return nil, false
	} else if startIdx == lastIdx+1 {
		return nil, true
	} else if endIdx > lastIdx+1 {
		endIdx = lastIdx + 1
	}
	var entries []raft.RaftEntry
	var idx = startIdx
	iter_cb := func(item *gkvlite.Item) bool {
		if idx >= endIdx {
			return false
		}
		if idx != U64Dec(item.Key) { // sanity check
			panic("Corrupted log!")
		}

		entry, err := LogValDec(item.Val)
		if err != nil {
			panic("Corrupted log entry!")
		}
		entries = append(entries, *entry)
		idx += 1
		return true
	}
	self.rlog.VisitItemsAscend(U64Enc(startIdx), true, iter_cb)
	return entries, true
}

func (self *SimplePster) LogUpdate(startIdx uint64, slice []raft.RaftEntry) bool {
	lastIdx := self.lastIdx()

	if (lastIdx == NilIdx && startIdx == 0) || (lastIdx+1 >= startIdx) {
		if len(slice) == 0 {
			return true // nothing to update
		}
		if lastIdx != NilIdx { // truncate
			newTailIdx := startIdx + uint64(len(slice)) - 1
			for idx := lastIdx; idx > newTailIdx; idx -= 1 {
				deleted, _ := self.rlog.Delete(U64Enc(idx))
				if !deleted {
					panic("Corrupt log!")
				}
			}
		}
		idx := startIdx
		for _, entry := range slice { // append/update
			blob, err := LogValEnc(&entry)
			if err != nil {
				panic("Impossible encode error!!")
			}
			err = self.rlog.Set(U64Enc(idx), blob)
			if err != nil {
				return false
			} // panic??
			idx += 1
		}
		return self.Sync()
	}
	return false
}

func (self *SimplePster) GetFields() *raft.RaftFields {
	blob, _ := self.rfields.Get([]byte{0})
	if blob == nil {
		return nil
	}
	return FieldsDec(blob)
}

func (self *SimplePster) SetFields(fields raft.RaftFields) bool {
	err := self.rfields.Set([]byte{0}, FieldsEnc(&fields))
	if err != nil {
		return false
	}
	return self.Sync()
}

func (self *SimplePster) Sync() bool {
	err := self.store.Flush()
	// No need to file.Sync() due to O_SYNC
	return err == nil
}

func NewPster(dbpath string, errlog *log.Logger) (*SimplePster, error) { // {{{1
	var store *gkvlite.Store
	file, err := os.OpenFile(dbpath, os.O_RDWR|os.O_CREATE|os.O_SYNC, 0660)
	if err != nil {
		return nil, err
	}
	store, err = gkvlite.NewStore(file)
	if err != nil {
		return nil, err
	}
	return &SimplePster{
		file:    file,
		store:   store,
		rlog:    store.SetCollection("rlog", nil),
		rfields: store.SetCollection("rfields", nil),
		err:     errlog,
	}, nil
}

func (self *SimplePster) Close() { // {{{1
	self.store.Close()
}
