package main

import (
    "errors"
    "math"
    "math/rand"
    "time"
    "github.com/davecheney/junk/clock"
)

type Action struct {
    Req Request
    Reply chan Response
}

type storeData struct {
    Version uint64 // if version is 0, a new version is automatically assigned
    ExpTime time.Time
    Contents []byte
}

type store map[string]*storeData

var FileNotFound = "FILE_NOT_FOUND"
var VersionMismatch = "VERSION"
var InternalError = "INTERNAL"

func InitStore() chan<- Action {
    ca := make(chan Action)
    go actionLoop(ca)
    return ca
}

func expiryTime(delaySecs uint64) time.Time {
    if delaySecs == 0 {
        return time.Unix(0, 0)
    } else {
        dur := time.Duration(delaySecs) * time.Second
        return clock.Monotonic.Now().Add(dur)
    }
}

func remainingSecs(t time.Time) (uint64, bool) {
    if t == time.Unix(0, 0) {
        return 0, true
    } else {
        rem := float64(t.Sub(clock.Monotonic.Now()))/float64(time.Second)
        if rem > 0.0 {
            return uint64(math.Ceil(rem)), true
        } else {
            return 0, false
        }
    }
}

func actionLoop(ca <-chan Action) {
    var s store = make(map[string]*storeData)
    for {
        action := <-ca
        var res Response
        switch req := action.Req.(type) {
        case *ReqRead:
            data := s.Get(req.FileName)
            if data == nil {
                res = &ResError { Desc: FileNotFound }
            } else {
                rem, _ := remainingSecs(data.ExpTime)
                res = &ResContents {
                    FileName: req.FileName,
                    Version: data.Version,
                    ExpTime: rem,
                    Contents: data.Contents,
                }
            }
        case *ReqWrite:
            ver := s.Set(req.FileName, &storeData {
                Version: 0,
                ExpTime: expiryTime(req.ExpTime),
                Contents: req.Contents,
            })
            res = &ResOkVer { Version: ver }
        case *ReqCaS:
            // use version 0 to write only if does not exist
            ver, err := s.CaS(req.FileName, &storeData {
                Version: req.Version,
                ExpTime: expiryTime(req.ExpTime),
                Contents: req.Contents,
            })
            if ver > 0 {
                res = &ResOkVer { Version: ver }
            } else {
                res = &ResError { Desc: err.Error() }
            }
        case *ReqDelete:
            if s.Unset(req.FileName) {
                res = &ResOk { }
            } else {
                res = &ResError { Desc: FileNotFound }
            }
        }
        action.Reply <- res
    }
}

func (s store) Get(key string) *storeData {
    value := s[key]
    if value != nil {
        _, ok := remainingSecs(value.ExpTime)
        if ok {
            return value
        } else {
            s.Unset(key)
            return nil
        }
    }
    return nil
}

func (s store) Version(key string) uint64 {
    value := s.Get(key)
    if value == nil {
        return 0
    } else {
        return value.Version
    }
}

func (s store) Set(key string, value *storeData) uint64 {
    // value.Version is ignored
    curver := s.Version(key)
    if curver == 0 {
        value.Version = uint64(rand.Uint32()) + 1
    } else {
        value.Version = curver + 1
    }
    s[key] = value
    return value.Version
}

func (s store) CaS(key string, value *storeData) (uint64, error) {
    // value.Version is matched with the current version; not threadsafe
    if value.Version == s.Version(key) {
        return s.Set(key, value), nil
    } else if s.Version(key) == 0 {
        return 0, errors.New(FileNotFound)
    } else {
        return 0, errors.New(VersionMismatch)
    }
}

func (s store) Unset(key string) bool {
    if value, ok := s[key]; ok {
        _, ok := remainingSecs(value.ExpTime)
        delete(s, key)
        if ok { return true } else { return false }
    } else {
        return false
    }
}
