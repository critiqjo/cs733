package main

import (
    "bufio"
    "fmt"
    "io"
    "net"
)

func handleClient(conn net.Conn) {
    rstream := bufio.NewReader(conn)
    wstream := bufio.NewWriter(conn)
    defer conn.Close()
    for {
        r, e := ParseRequest(rstream)
        if r != nil {
            _, e = wstream.WriteString("OK\r\n")
        } else if e != io.EOF {
            _, e = wstream.WriteString("ERR_CMD_ERR\r\n")
        } else { break }
        if e != nil { break }
        wstream.Flush()
    }
}

func main() {
    ln, err := net.Listen("tcp", ":8080")
    if err != nil {
        return
    }

    fmt.Println("Waiting for clients!")
    for {
        conn, err := ln.Accept()
        if err != nil {
            fmt.Println(err)
            break
        }
        go handleClient(conn)
    }
}
