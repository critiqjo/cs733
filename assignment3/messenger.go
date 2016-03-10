package main

import (
    "bufio"
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
    "sync"
    "time"
)

type SimpleMsger struct {
    nodeId  uint32
    raftCh  chan<- raft.Message
    pListen mangos.Socket
    peers   map[uint32]mangos.Socket
    pCAddr  map[uint32]string // peer's client socket address map
    cListen net.Listener
    cRespCh *cRespChanMap
    cRespTO time.Duration // response timeout
    err     *log.Logger
}

type cRespChanMap struct { // {{{1
    sync.Mutex
    inner map[uint64]chan<- string // uid -> response channel
}

func newCRespChanMap() *cRespChanMap {
    return &cRespChanMap { inner: make(map[uint64]chan<- string) }
}

func (self *cRespChanMap) insert(key uint64, ch chan<- string) {
    self.Lock()
    self.inner[key] = ch
    self.Unlock()
}

func (self *cRespChanMap) remove(key uint64) (chan<- string, bool) {
    self.Lock()
    ch, ok := self.inner[key]
    delete(self.inner, key)
    self.Unlock()
    return ch, ok
}

type Node struct { // {{{1
    Host    string  `json:"host-ip"`
    PPort   int     `json:"peer-port"`
    CPort   int     `json:"client-port"`
}

func NewMsger(nodeId uint32, cluster map[uint32]Node) (*SimpleMsger, error) { // {{{1
    var mangosock mangos.Socket
    var err error
    if mangosock, err = pull.NewSocket(); err != nil {
        return nil, err
    }
    mangosock.AddTransport(tcp.NewTransport())
    node, ok := cluster[nodeId]
    if !ok { return nil, errors.New("nodeId not in cluster") }
    listenAddr := fmt.Sprintf("tcp://%v:%v", node.Host, node.PPort)
    if err = mangosock.Listen(listenAddr); err != nil {
        return nil, err
    }

    var peers = make(map[uint32]mangos.Socket)
    var redirs = make(map[uint32]string)
    for peerId, peerNode := range cluster {
        if peerId != nodeId {
            var sock mangos.Socket
            if sock, err = push.NewSocket(); err != nil {
                return nil, err
            }
            peerAddr := fmt.Sprintf("tcp://%v:%v", peerNode.Host, peerNode.PPort)
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
        raftCh:  nil,
        pListen: mangosock,
        peers:   peers,
        pCAddr:  redirs,
        cListen: netsock,
        cRespCh: newCRespChanMap(),
        cRespTO: 30 * time.Second,
        err:     log.New(os.Stderr, "-- ", log.Lshortfile),
    }, nil
}

// ---- quack like a Messenger {{{1
func (self *SimpleMsger) Register(raftCh chan<- raft.Message) {
    self.raftCh = raftCh
}

func (self *SimpleMsger) Send(nodeId uint32, msg raft.Message) {
    if sock, ok := self.peers[nodeId]; ok {
        data, err := MsgEnc(msg)
        if err == nil {
            if err := sock.Send(data); err != nil {
                self.err.Print(err)
            }
        } else {
            self.err.Print(err)
        }
    } else {
        self.err.Print("Bad nodeId")
    }
}

func (self *SimpleMsger) BroadcastVoteRequest(msg *raft.VoteRequest) {
    for nodeId, _ := range self.peers {
        self.Send(nodeId, msg)
    }
}

func (self *SimpleMsger) Client301(uid uint64, nodeId uint32) {
    self.RespondToClient(uid, fmt.Sprintf("ERR301 %v", self.pCAddr[nodeId]))
}

func (self *SimpleMsger) Client503(uid uint64) {
    self.RespondToClient(uid, "ERR503")
}

func (self *SimpleMsger) SpawnListeners() { // {{{1
    go self.listenToPeers()
    go self.listenToClients()
}

func (self *SimpleMsger) listenToPeers() {
    for {
        data, err := self.pListen.Recv()
        if err != nil {
            self.err.Print("Fatal error:", err)
            break
        }
        msg, err := MsgDec(data)
        self.err.Print("Received ", msg)
        if err == nil {
            self.raftCh <- msg
        } else {
            self.err.Print(err)
        }
    }
}

func (self *SimpleMsger) listenToClients() {
    for {
        conn, err := self.cListen.Accept()
        if err != nil {
            self.err.Print("Fatal error:", err)
            break
        }
        go self.handleClient(conn)
    }
}

func (self *SimpleMsger) RespondToClient(uid uint64, msg string) { // {{{1
    if respCh, ok := self.cRespCh.remove(uid); ok {
        respCh <- msg // client timeout could happen in parallel
    }
}

func (self *SimpleMsger) handleClient(conn net.Conn) { // {{{1
    rstream := bufio.NewReader(conn)
    wstream := bufio.NewWriter(conn)

    respond := func(resp string) bool {
        if _, err := wstream.WriteString(resp + "\r\n"); err != nil { return false }
        if    err := wstream.Flush();                    err != nil { return false }
        return true
    }
    respCh := make(chan string, 1)
    defer conn.Close()
    if self.raftCh == nil {
        _, _ = wstream.WriteString("ERR503\r\n")
        return
    }
    for {
        // FIXME have a read deadline?
        ce, eof := ParseCEntry(rstream)
        if ce != nil {
            self.cRespCh.insert(ce.UID, respCh)
            self.raftCh <- ce
            var resp string
            select {
            case resp = <-respCh:
            case <-time.After(self.cRespTO): // timeout
                resp = "ERR504"
                self.cRespCh.remove(ce.UID)
            }
            if ok := respond(resp); !ok { break }
        } else if !eof {
            if ok := respond("ERR400"); !ok { break }
        } else {
            break
        }
    }
}
