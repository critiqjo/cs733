package main

import (
    "errors"
    "math/rand"
)

type Action struct {
    Req Request
    Reply chan Response
}

type storeData struct {
    Version uint64 // if version is 0, a new version is automatically assigned
    ExpTime uint64
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
                res = &ResContents {
                    FileName: req.FileName,
                    Version: data.Version,
                    ExpTime: data.ExpTime,
                    Contents: data.Contents,
                }
            }
        case *ReqWrite:
            ver := s.Set(req.FileName, &storeData {
                Version: 0,
                ExpTime: req.ExpTime,
                Contents: req.Contents,
            })
            res = &ResOkVer { Version: ver }
        case *ReqCaS:
            // use version 0 to write only if does not exist
            ver, err := s.CaS(req.FileName, &storeData {
                Version: req.Version,
                ExpTime: req.ExpTime,
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
    return s[key] // nil if does not exist
}

func (s store) Version(key string) uint64 {
    if s[key] == nil {
        return 0
    } else {
        return s[key].Version
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
    if _, ok := s[key]; ok {
        delete(s, key)
        return true
    } else {
        return false
    }
}
