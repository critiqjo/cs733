package main

type Action struct {
    Req Request
    Reply chan Response
}

func actionLoop(ca chan Action) {
    for {
        action := <-ca
        var res Response
        switch req := action.Req.(type) {
        case *ReqRead:
            res = &ResContents {
                FileName: req.FileName,
                Version: 0,
                Size: 5,
                ExpTime: 0,
                Contents: []byte("dummy"),
            }
        case *ReqWrite:
            res = &ResOkVer { Version: 0 }
        case *ReqCaS:
            res = &ResOkVer { Version: 0 }
        case *ReqDelete:
            res = &ResOk { }
        }
        action.Reply <- res
    }
}

func InitStore() chan Action {
    ca := make(chan Action)
    go actionLoop(ca)
    return ca
}
