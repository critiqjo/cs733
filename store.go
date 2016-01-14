package main

type Action struct {
    req Request
    reply chan Response
}
