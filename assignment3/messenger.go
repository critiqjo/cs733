package main

import (
    "errors"
    "fmt"
    "log"
    "github.com/critiqjo/cs733/assignment3/raft"
    "github.com/go-mangos/mangos"
    "github.com/go-mangos/mangos/protocol/pull"
    "github.com/go-mangos/mangos/protocol/push"
    "github.com/go-mangos/mangos/transport/tcp"
    "net"
    "os"
)

type SimpleMsger struct {
    nodeId  int
    rListen mangos.Socket
    cListen net.Listener
    peers   map[int]mangos.Socket
    redirs  map[int]string // client-redirect addresses
    raftch  chan<- raft.Message
    err     *log.Logger
}

type Node struct {
    Host    string
    RPort   int
    CPort   int
}

func NewMsger(nodeId int, cluster map[int]Node) (*SimpleMsger, error) { // {{{1
    var mangosock mangos.Socket
    var err error
    if mangosock, err = pull.NewSocket(); err != nil {
        return nil, err
    }
    mangosock.AddTransport(tcp.NewTransport())
    node, ok := cluster[nodeId]
    if !ok { return nil, errors.New("nodeId not in cluster") }
    listenAddr := fmt.Sprintf("tcp://%v:%v", node.Host, node.RPort)
    if err = mangosock.Listen(listenAddr); err != nil {
        return nil, err
    }

    var peers = make(map[int]mangos.Socket)
    var redirs = make(map[int]string)
    for peerId, peerNode := range cluster {
        if peerId != nodeId {
            var sock mangos.Socket
            if sock, err = push.NewSocket(); err != nil {
                return nil, err
            }
            peerAddr := fmt.Sprintf("tcp://%v:%v", peerNode.Host, peerNode.RPort)
            sock.AddTransport(tcp.NewTransport())
            if err = sock.Dial(peerAddr); err != nil {
                return nil, err
            }
            peers[peerId] = sock
            redirs[peerId] = fmt.Sprintf("%v:%v", peerNode.Host, peerNode.CPort)
        }
    }

    var netsock net.Listener
    netsock, err = net.Listen("tcp", fmt.Sprintf(":%v", node.CPort))
    if err != nil {
        return nil, err
    }

    return &SimpleMsger {
        nodeId:  nodeId,
        rListen: mangosock,
        cListen: netsock,
        peers:   peers,
        redirs:  redirs,
        raftch:  nil,
        err:     log.New(os.Stderr, "-- ", log.Lshortfile),
    }, nil
}

// quack like a Messenger {{{1
func (self *SimpleMsger) Register(raftch chan<- raft.Message) {
    self.raftch = raftch
}

func (self *SimpleMsger) Send(nodeId int, msg raft.Message) {
    if sock, ok := self.peers[nodeId]; ok {
        if err := sock.Send(Encode(msg)); err != nil {
            self.err.Print(err)
        }
    }
}

func (self *SimpleMsger) BroadcastVoteRequest(msg *raft.VoteRequest) {
    for _, sock := range self.peers {
        if err := sock.Send(Encode(msg)); err != nil {
            self.err.Print(err)
        }
    }
}

func (self *SimpleMsger) Client301(uid uint64, nodeId int) {
}

func (self *SimpleMsger) Client503(uid uint64) {
}
