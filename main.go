package main

import (
    "bufio"
    "fmt"
    "io"
    "net"
)

func main() {
    ln, err := net.Listen("tcp", ":8080")
    if err != nil {
        return
    }
    conn, err := ln.Accept()
    if err != nil {
        return
    }
    rstream := bufio.NewReader(conn)
    for {
        p, e := ParseRequest(rstream)
        if p != nil {
            fmt.Println("Yay!")
        } else {
            fmt.Println("No!")
            if e == io.EOF {
                break
            }
        }
    }
}
