package main

import (
	"encoding/json"
	"fmt"
	"github.com/critiqjo/cs733/assignment4/raft"
	"log"
	"os"
	"strconv"
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
	errlog := log.New(os.Stderr, "-- ", log.Lshortfile) // | log.Lmicroseconds

	msger, err := NewMsger(uint32(selfId), cluster, errlog)
	if err != nil {
		fmt.Printf("Error creating messenger: %v\n", err.Error())
		os.Exit(1)
	}
	pster, err := NewPster(logfile, errlog)
	if err != nil {
		fmt.Printf("Error creating persister: %v\n", err.Error())
		os.Exit(1)
	}
	machn := NewMachn(0, msger)

	node, err := raft.NewNode(uint32(selfId), nodeIds, 16, msger, pster, machn, errlog)
	if err != nil {
		fmt.Printf("Error creating raft node: %v\n", err.Error())
		os.Exit(1)
	}

	msger.SpawnListeners()
	node.Run(time.Duration(200) * time.Millisecond)
}
