package main

import (
	"fmt"
	"github.com/critiqjo/cs733/assignment4/raft"
	"github.com/critiqjo/cs733/assignment4/store"
)

type SimpleMachn struct {
	storeChan chan<- store.Action
	respCache map[uint64]string // uid -> response
	msger     *SimpleMsger
}

// ---- quack like a Machine {{{1
func (self *SimpleMachn) Execute(centries []raft.ClientEntry) {
	resChan := make(chan store.Response)
	for _, cEntry := range centries {
		action := store.Action{
			Req:   cEntry.Data,
			Reply: resChan,
		}
		self.storeChan <- action
		response := <-resChan

		switch r := response.(type) {
		case *store.ResOk:
			self.respCache[cEntry.UID] = "OK"
		case *store.ResOkVer:
			self.respCache[cEntry.UID] = fmt.Sprintf("OK %d", r.Version)
		case *store.ResContents:
			self.respCache[cEntry.UID] = fmt.Sprintf("CONTENTS %d %d %d\r\n%s",
				r.Version, len(r.Contents), r.ExpTime, string(r.Contents))
		case *store.ResError:
			self.respCache[cEntry.UID] = fmt.Sprintf("%s", r.Desc)
		}
		_ = self.TryRespond(cEntry.UID)
	}
}

func (self *SimpleMachn) TryRespond(uid uint64) bool {
	if resp, ok := self.respCache[uid]; ok {
		self.msger.RespondToClient(uid, resp)
		return true
	} else {
		return false
	}
}

func NewMachn(initState int64, msger *SimpleMsger) *SimpleMachn { // {{{1
	storeChan := store.InitStore()
	return &SimpleMachn{
		storeChan: storeChan,
		respCache: make(map[uint64]string),
		msger:     msger,
	}
}
