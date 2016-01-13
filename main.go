package main

import (
    "bufio"
    "fmt"
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
        p := ParseRequest(rstream)
        if p != nil {
            fmt.Println("Yay!")
        } else {
            fmt.Println("No!")
            break
        }
    }
}
