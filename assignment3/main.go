package main

import (
    "encoding/json"
    "fmt"
    "github.com/critiqjo/cs733/assignment3/raft"
    "math/rand"
    "strconv"
    "os"
    "time"
)

func main() {
    args := os.Args
    if len(args) != 4 {
        fmt.Printf("Usage: %v <cluster-file> <log-file> <node-id>\n", args[0])
        os.Exit(1)
    }

    file, err := os.Open(args[1])
    if err != nil {
        fmt.Printf("Error reading cluster config file: %v\n", err.Error())
        os.Exit(1)
    }

    dec := json.NewDecoder(file)
    var cluster_json map[string]Node
    err = dec.Decode(&cluster_json)
    if err != nil {
        fmt.Printf("Error parsing json file: %v\n", err.Error())
        os.Exit(1)
    }

    selfId, err := strconv.ParseUint(args[3], 10, 32)
    if err != nil {
        fmt.Printf("Error parsing node-id: %v\n", err.Error())
        os.Exit(1)
    }

    var cluster = make(map[uint32]Node)
    var nodeIds []uint32
    for nodeIdStr, nodeConf := range cluster_json {
        nodeId, err := strconv.ParseUint(nodeIdStr, 10, 32)
        if err != nil {
            fmt.Printf("Error parsing node-ids in json file: %v\n", err.Error())
            os.Exit(1)
        }
        cluster[uint32(nodeId)] = nodeConf
        nodeIds = append(nodeIds, uint32(nodeId))
    }

    logfile := args[2]

    msger, err := NewMsger(uint32(selfId), cluster)
    if err != nil {
        fmt.Printf("Error creating messenger: %v\n", err.Error())
        os.Exit(1)
    }
    pster, err := NewPster(logfile)
    if err != nil {
        fmt.Printf("Error creating persister: %v\n", err.Error())
        os.Exit(1)
    }
    machn := NewMachn(0, msger)

    node, err := raft.NewNode(uint32(selfId), nodeIds, 16, msger, pster, machn)
    if err != nil {
        fmt.Printf("Error creating raft node: %v\n", err.Error())
        os.Exit(1)
    }

    InitCoder()

    msger.SpawnListeners()
    node.Run(func(state raft.RaftState) time.Duration {
        switch state {
        case raft.Follower:
            return time.Duration(400 + rand.Int63n(200)) * time.Millisecond
        case raft.Candidate:
            return time.Duration(600 + rand.Int63n(300)) * time.Millisecond
        case raft.Leader:
            return time.Duration(200 + rand.Int63n(100)) * time.Millisecond
        }
        panic("Unreachable")
    })
}
