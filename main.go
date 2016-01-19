package main

import (
    "bufio"
    "fmt"
    "io"
    "net"
)

func handleClient(conn net.Conn, storeChan chan<- Action) {
    rstream := bufio.NewReader(conn)
    wstream := bufio.NewWriter(conn)
    defer conn.Close()
    resChan := make(chan Response)
    for {
        req, err := ParseRequest(rstream)
        if req != nil {
            action := Action {
                Req: req,
                Reply: resChan,
            }
            storeChan <- action
            response := <-resChan
            var resHead string
            var resBody []byte = nil

            switch r := response.(type) {
            case *ResOk:
                resHead = "OK\r\n"
            case *ResOkVer:
                resHead = fmt.Sprintf("OK %d\r\n", r.Version)
            case *ResContents:
                resHead = fmt.Sprintf("CONTENTS %d %d %d\r\n",
                                      r.Version, len(r.Contents), r.ExpTime)
                resBody = r.Contents
            case *ResError:
                resHead = fmt.Sprintf("ERR_%s\r\n", r.Desc)
            }

            _, err = wstream.WriteString(resHead)
            if err == nil && resBody != nil {
                _, err = wstream.Write(resBody)
                if err == nil {
                    _, err = wstream.WriteString("\r\n")
                }
            }
        } else if err != io.EOF {
            _, err = wstream.WriteString("ERR_CMD_ERR\r\n")
            // break on any error?
        }

        if err != nil { break } // EOF or writing failed
        wstream.Flush()
    }
}

func main() {
    ln, err := net.Listen("tcp", ":8080")
    if err != nil {
        fmt.Println(err)
        return
    }

    storeChan := InitStore()
    for {
        conn, err := ln.Accept()
        if err != nil {
            fmt.Println(err)
            break
        }
        go handleClient(conn, storeChan)
    }
}
