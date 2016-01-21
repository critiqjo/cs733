package main

import (
    "bufio"
    "errors"
    "io"
    "regexp"
    "strconv"
)

func ParseRequest(rstream *bufio.Reader) (Request, error) {
    // FileName is assumed to have no whitespace characters including \r and \n
    str, err := rstream.ReadString('\n')
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
        size, _ := strconv.Atoi(matches[2])
        var exp uint64 = 0
        if len(matches[3]) > 0 {
            exp, _ = strconv.ParseUint(matches[3], 10, 64)
        }
        contents, err := getContentsSkipCRLF(rstream, size)
        if err != nil { return nil, err }
        return &ReqWrite { FileName: matches[1],
                           ExpTime: exp,
                           Contents: contents,
                         }, nil
    }

    pat = regexp.MustCompile("^cas ([^ ]+) ([0-9]+) ([0-9]+) ?([0-9]+)?\r\n$")
    matches = pat.FindStringSubmatch(str)
    if len(matches) > 0 {
        ver, _ := strconv.ParseUint(matches[2], 10, 64)
        size, _ := strconv.Atoi(matches[3])
        var exp uint64 = 0
        if len(matches[4]) > 0 {
            exp, _ = strconv.ParseUint(matches[4], 10, 64)
        }
        contents, err := getContentsSkipCRLF(rstream, size)
        if err != nil { return nil, err }
        return &ReqCaS { FileName: matches[1],
                         Version: ver,
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

func getContentsSkipCRLF(rstream *bufio.Reader, size int) ([]byte, error) {
    // Reads size bytes, consumes until LF is found (trail), then matches the
    // trail with CRLF. Returns size bytes if everything went well, else nil
    contents := make([]byte, size)
    _, err := io.ReadFull(rstream, contents)
    if err != nil {
        return nil, err
    }
    trail, err := rstream.ReadSlice('\n')
    if err != nil {
        return nil, err
    } else if len(trail) != 2 || trail[0] != '\r' || trail[1] != '\n' {
        return nil, errors.New("Contents did not end in CRLF")
    }
    return contents, nil
}
