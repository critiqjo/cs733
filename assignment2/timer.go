package main

import "time"

type RaftTimer struct {
    version uint64
    funcGen func(uint64) func()
    sampler func(RaftState) time.Duration
    t *time.Timer
}

func NewRaftTimer(ff func(uint64) func(), tf func(RaftState) time.Duration) *RaftTimer {
    return &RaftTimer { 0, ff, tf, nil }
}

func (self *RaftTimer) Reset(rs RaftState) {
    dur := self.sampler(rs)
    if self.t == nil || !self.t.Reset(dur) {
        self.version += 1
        self.t = time.AfterFunc(dur, self.funcGen(self.version))
    }
}

func (self *RaftTimer) Match(v uint64) bool {
    return self.version == v
}
