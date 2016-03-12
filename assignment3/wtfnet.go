package main

import (
    "bufio"
    "errors"
    "io"
    "net"
    "time"
)

type WtfPush struct { // {{{1
    addr *net.TCPAddr
    conn *net.TCPConn
    pushch chan []byte
}

func NewWtfPush(straddr string) (*WtfPush, error) {
    addr, err := net.ResolveTCPAddr("tcp", straddr)
    if err != nil { return nil, err }
    return &WtfPush {
        addr: addr,
        conn: nil,
        pushch: make(chan []byte),
    }, nil
}

// silently discards on error
func (self *WtfPush) Push(blob []byte) {
    select {
    case self.pushch <- blob:
    default:
    }
}

// the push loop (forever and ever)
func (self *WtfPush) Run() {
    for {
        blob := <-self.pushch
        if self.conn == nil {
            conn, err := net.DialTCP("tcp", nil, self.addr)
            if err == nil {
                self.conn = conn
            } else {
                continue
            }
        }
        if self.conn != nil {
            err := SendBlob(self.conn, blob)
            if err != nil {
                _ = self.conn.Close()
                self.conn = nil
            }
        }
    }
}

func WriteHard(conn net.Conn, blob []byte) error { // {{{1
    var nn int = 0
    for nn < len(blob) {
        n, err := conn.Write(blob[nn:])
        if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
            time.Sleep(50 * time.Millisecond)
            nn += n
            continue
        } else {
            return err
        }
    }
    return nil
}

func SendBlob(conn net.Conn, blob []byte) error { // {{{1
    head := append(U64Enc(uint64(len(blob))), '\r', '\n')
    err := WriteHard(conn, head)
    if err != nil { return err }
    err = WriteHard(conn, blob)
    if err != nil { return err }
    return nil
}

func RecvBlob(rstream *bufio.Reader) ([]byte, error) { // {{{1
    head, err := ReadExactly(rstream, 10)
    if err != nil { return nil, err }
    if head[8] != '\r' && head[9] != '\n' {
        return nil, errors.New("Bad header!")
    }
    size := U64Dec(head[:8])
    if size > 64e6 { // 64 MB
        return nil, errors.New("Object size too high!!")
    }
    body, err := ReadExactly(rstream, int(size))
    if err != nil { return nil, err }
    return body, nil
}

func ReadExactly(rstream *bufio.Reader, size int) ([]byte, error) { // {{{1
    if size < 0 { return nil, errors.New("Negative size?!") }
    contents := make([]byte, uint64(size))
    _, err := io.ReadFull(rstream, contents)
    if err != nil {
        return nil, err
    }
    return contents, nil
}

// Note: single occurrence of '\r' is ok, but that of '\n' is an error
func ReadLineClean(rstream *bufio.Reader) (string, error) { // {{{1
    line, err := rstream.ReadString('\n')
    if err != nil { return "", err }
    crIdx := len(line) - 2
    if crIdx >= 0 && line[crIdx] == '\r' {
        return line[:crIdx], nil
    } else {
        return "", errors.New("Bad format")
    }
}
