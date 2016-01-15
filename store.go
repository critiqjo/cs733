package main

type Action struct {
    Req Request
    Reply chan Response
}

type storeData struct {
    Version uint64
    ExpTime uint64
    Contents []byte
}

type store map[string]*storeData

func InitStore() chan Action {
    ca := make(chan Action)
    go actionLoop(ca)
    return ca
}

func actionLoop(ca chan Action) {
    var s store = make(map[string]*storeData)
    for {
        action := <-ca
        var res Response
        switch req := action.Req.(type) {
        case *ReqRead:
            data := s.Get(req.FileName)
            if data == nil {
                res = &ResError {
                    Desc: "FILE_NOT_FOUND",
                }
            } else {
                res = &ResContents {
                    FileName: req.FileName,
                    Version: data.Version,
                    ExpTime: data.ExpTime,
                    Contents: data.Contents,
                }
            }
        case *ReqWrite:
            s.Set(req.FileName, &storeData {
                Version: 1,
                ExpTime: req.ExpTime,
                Contents: req.Contents,
            })
            res = &ResOkVer { Version: 0 }
        case *ReqCaS:
            res = &ResOkVer { Version: 0 }
        case *ReqDelete:
            res = &ResOk { }
        }
        action.Reply <- res
    }
}

func (s store) Get(key string) *storeData {
    if val, ok := s[key]; ok {
        return val
    } else {
        return nil
    }
}

func (s store) Set(key string, value *storeData) {
    s[key] = value
}
