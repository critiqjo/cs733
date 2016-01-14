package main

import (
    "bufio"
    "errors"
    "regexp"
    "strconv"
)

func ParseRequest(stream *bufio.Reader) (Request, error) {
    // FileName is assumed to have no whitespace characters including \r and \n
    str, err := stream.ReadString('\n')
    if err != nil { // if and only if str does not end in '\n'
        return nil, err
    }

    pat := regexp.MustCompile("^read ([^ ]+)\r\n$")
    matches := pat.FindStringSubmatch(str)
    if len(matches) > 0 {
        return &ReqRead { FileName: matches[1] }, nil
    }

    pat = regexp.MustCompile("^write ([^ ]+) ([0-9]+) ?([0-9]+)?\r\n$")
    matches = pat.FindStringSubmatch(str)
    if len(matches) > 0 {
        size, _ := strconv.ParseUint(matches[2], 10, 64)
        var exp uint64 = 0
        if len(matches[3]) > 0 {
            exp, _ = strconv.ParseUint(matches[3], 10, 64)
        }
        contents, err := consumeWithCRLF(stream, size)
        if err != nil { return nil, err }
        return &ReqWrite { FileName: matches[1],
                           Size: size,
                           ExpTime: exp,
                           Contents: contents,
                         }, nil
    }

    pat = regexp.MustCompile("^cas ([^ ]+) ([0-9]+) ([0-9]+) ?([0-9]+)?\r\n$")
    matches = pat.FindStringSubmatch(str)
    if len(matches) > 0 {
        ver, _ := strconv.ParseUint(matches[2], 10, 64)
        size, _ := strconv.ParseUint(matches[3], 10, 64)
        var exp uint64 = 0
        if len(matches[4]) > 0 {
            exp, _ = strconv.ParseUint(matches[4], 10, 64)
        }
        contents, err := consumeWithCRLF(stream, size)
        if err != nil { return nil, err }
        return &ReqCaS { FileName: matches[1],
                         Version: ver,
                         Size: size,
                         ExpTime: exp,
                         Contents: contents,
                       }, nil
    }

    pat = regexp.MustCompile("^delete ([^ ]+)\r\n$")
    matches = pat.FindStringSubmatch(str)
    if len(matches) > 0 {
        return &ReqDelete { FileName: matches[1] }, nil
    }

    return nil, errors.New("Invalid format!")
}

func consumeWithCRLF(stream *bufio.Reader, size uint64) ([]byte, error) {
    contents := make([]byte, size)
    var readsize uint64 = 0
    var n int = 0
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
