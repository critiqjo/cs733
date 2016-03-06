package main

import (
    "errors"
    "github.com/critiqjo/cs733/assignment3/raft"
    "github.com/go-mangos/mangos"
    "github.com/go-mangos/mangos/protocol/pull"
    "github.com/go-mangos/mangos/protocol/push"
    "github.com/go-mangos/mangos/transport/tcp"
)

type SimpleMsger struct {
    nodeId int
    peers  map[int]mangos.Socket
    sock   mangos.Socket
    raftch chan<- raft.Message
}

func NewMsger(nodeId int, cluster map[int]string) (*SimpleMsger, error) { // {{{1
    var sock mangos.Socket
    var err error
    if sock, err = pull.NewSocket(); err != nil {
        return nil, err
    }
    sock.AddTransport(tcp.NewTransport())
    listenAddr, ok := cluster[nodeId]
    if !ok { return nil, errors.New("nodeId not in cluster") }
    if err = sock.Listen(listenAddr); err != nil {
        return nil, err
    }

    var peers = make(map[int]mangos.Socket)
    for peerId, peerAddr := range cluster {
        if peerId != nodeId {
            var sock mangos.Socket
            var err error
            if sock, err = push.NewSocket(); err != nil {
                return nil, err
            }
            sock.AddTransport(tcp.NewTransport())
            if err = sock.Dial(peerAddr); err != nil {
                return nil, err
            }
            peers[peerId] = sock
        }
    }

    return &SimpleMsger {
        nodeId: nodeId,
        peers: peers,
        sock: sock,
        raftch: nil,
    }, nil
}

// quack like a Messenger {{{1
func (self *SimpleMsger) Register(raftch chan<- raft.Message) {
    self.raftch = raftch
}

func (self *SimpleMsger) Send(nodeId int, msg raft.Message) {
}

func (self *SimpleMsger) BroadcastVoteRequest(msg *raft.VoteRequest) {
}

func (self *SimpleMsger) Client301(uid uint64, nodeId int) {
}

func (self *SimpleMsger) Client503(uid uint64) {
}
