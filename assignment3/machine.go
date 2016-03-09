package main

import (
    "github.com/critiqjo/cs733/assignment3/raft"
    "strconv"
)

// State machine commands
type MachnRead struct {}
type MachnUpdate struct {
    Value int64
}

type SimpleMachn struct {
    state int64
    respCache map[uint64]int64 // uid -> state map
    msger *SimpleMsger
}

// ---- quack like a Machine {{{1
func (self *SimpleMachn) Execute(centries []raft.ClientEntry) {
    for _, cEntry := range centries {
        switch action := cEntry.Data.(type) {
        case *MachnRead:
        case *MachnUpdate:
            self.state = action.Value - self.state
        default:
            panic("Invalid action")
        }
        self.respCache[cEntry.UID] = self.state
        _ = self.TryRespond(cEntry.UID)
    }
}

func (self *SimpleMachn) TryRespond(uid uint64) bool {
    if resp, ok := self.respCache[uid]; ok {
        self.msger.RespondToClient(uid, "OK " + strconv.FormatInt(resp, 10))
        return true
    } else {
        return false
    }
}

func NewMachn(initState int64, msger *SimpleMsger) *SimpleMachn { // {{{1
    return &SimpleMachn {
        state: initState,
        respCache: make(map[uint64]int64),
        msger: msger,
    }
}
