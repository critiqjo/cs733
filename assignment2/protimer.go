package main

import "time"

type ProTimer struct {
    version uint64
    funcGen func(uint64) func()
    sampler func() time.Duration
    t *time.Timer
}

func NewProTimer(ff func(uint64) func(), tf func() time.Duration) *ProTimer {
    return &ProTimer { 0, ff, tf, nil }
}

func (self *ProTimer) Reset() {
    dur := self.sampler()
    if self.t == nil || !self.t.Reset(dur) {
        self.version += 1
        self.t = time.AfterFunc(dur, self.funcGen(self.version))
    }
}

func (self *ProTimer) Match(v uint64) bool {
    return self.version == v
}
