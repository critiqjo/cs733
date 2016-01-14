package main

type Request interface {
    RequestDummy()
}

type ReqRead struct {
    FileName string
}
func (r *ReqRead) RequestDummy() {}

type ReqWrite struct {
    FileName string
    Size uint64
    ExpTime uint64
    Contents []byte
}
func (r *ReqWrite) RequestDummy() {}

type ReqCaS struct {
    FileName string
    Version uint64
    Size uint64
    ExpTime uint64
    Contents []byte
}
func (r *ReqCaS) RequestDummy() {}

type ReqDelete struct {
    FileName string
}
func (r *ReqDelete) RequestDummy() {}

type Response interface {
    ResponseDummy()
}

type ResOk struct { }
func (r *ResOk) ResponseDummy() {}

type ResContents struct {
    FileName string
    Version uint64
    Size uint64
    ExpTime uint64
    Contents []byte
}
func (r *ResContents) ResponseDummy() {}

type ResOkVer struct {
    Version uint64
}
func (r *ResOkVer) ResponseDummy() {}

type ResError struct {
    desc string
}
func (r *ResError) ResponseDummy() {}

// missing Rust's enum :(
