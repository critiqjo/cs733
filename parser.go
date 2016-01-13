package main

import (
    "bufio"
    "regexp"
)

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

func ParseRequest(stream *bufio.Reader) Request {
    // FileName is assumed to have no whitespace characters including \r and \n
    str, err := stream.ReadString('\n')
    if err != nil {
        return nil
    }

    pat := regexp.MustCompile("^read ([^ \r\n]+)\r\n$")
    matches := pat.FindStringSubmatch(str)
    if len(matches) > 0 {
        return &ReqRead { FileName: matches[1] }
    }

    return nil
}
