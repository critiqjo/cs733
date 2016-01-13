package main

import (
    "bufio"
    "errors"
    "regexp"
    "strconv"
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

    pat = regexp.MustCompile("^write ([^ \r\n]+) ([0-9]+) ?([0-9]+)?\r\n$")
    matches = pat.FindStringSubmatch(str)
    if len(matches) > 0 {
        size, err := strconv.ParseUint(matches[2], 10, 64)
        if err != nil { return nil }
        var exp uint64
        exp = 0
        if len(matches[3]) > 0 {
            exp, err = strconv.ParseUint(matches[3], 10, 64)
            if err != nil { return nil }
        }
        contents, err := consumeWithCRLF(stream, size)
        if err != nil { return nil }
        return &ReqWrite { FileName: matches[1],
                           Size: size,
                           ExpTime: exp,
                           Contents: contents,
                         }
    }

    return nil
}

func consumeWithCRLF(stream *bufio.Reader, size uint64) ([]byte, error) {
    contents := make([]byte, size)
    var readsize uint64
    var n int
    var e error
    for readsize < size {
        if n == 0 && e != nil {
            return nil, e
        }
        slice := contents[readsize:]
        n, e = stream.Read(slice)
        readsize += uint64(n)
    }
    l, _ := stream.ReadSlice('\n')
    if len(l) != 2 || l[0] != '\r' || l[1] != '\n' {
        return nil, errors.New("Bad ending!")
    }
    return contents, nil
}
