package main

import "time"

type rState int

const (
    follower rState = iota
    candidate
    leader
)

const N = 5 // number of servers -- should be part of rConsensus once
            // configuration changes support is implemented

type rConsensus struct {
    id int
    // persistent fields
    log []RaftLogEntry
    term uint64
    votedFor int
    // volatile fields
    state rState
    commitIdx uint64
    lastAppld uint64
    // state-specific fields
    voteCount int // candidate
    nextIdx []uint64 // leader
    matchIdx []uint64 // leader
}

type RaftNode struct {
    msger Messenger
    pster Persister
    machn Machine
    notifch chan Message
    timer *ProTimer
    consensus rConsensus
}

func NewRaftNode(serverId int, msger Messenger, pster Persister, machn Machine) RaftNode {
    ps := pster.StateRead()
    var term uint64
    var votedFor int
    if ps == nil {
        term = 0
        votedFor = -1
    } else {
        term = ps.T
        votedFor = ps.V
    }
    notifch := make(chan Message)
    msger.Register(notifch)
    log := pster.LogRead()
    if log == nil {
        // simplification: to avoid a few checks for empty log
        log = []RaftLogEntry { RaftLogEntry { 0, 0, nil } }
    }
    return RaftNode {
        msger, pster, machn, notifch, nil,
        rConsensus {
            id: serverId,
            log: log,
            term: term,
            votedFor: votedFor,
            state: follower,
            commitIdx: 0,
            lastAppld: 0, // TODO init using machn.LoadSnapshot()
            voteCount: 0,
            nextIdx: nil,
            matchIdx: nil,
        },
    }
}

// event loop, waits on notifch, and for timeouts
func (self *RaftNode) Run(timeoutSampler func() time.Duration) {
    self.timer = NewProTimer(func(v uint64) func() {
        return func() {
            self.notifch <- &timeout { v }
        }
    }, timeoutSampler)
    self.timer.Reset()
    for {
        msg := <-self.notifch
        switch self.consensus.state {
        case follower:
            self.followerHandler(msg)
        case candidate:
            self.candidateHandler(msg)
        case leader:
            self.leaderHandler(msg)
        }
        break
    }
}

func (self *RaftNode) followerHandler(m Message) {
    switch msg := m.(type) {
    case *AppendEntries:
        break
    case *RequestVote:
        break
    case *AppendEntriesResp:
        break
    case *RequestVoteResp:
        break
    case *ClientRequest:
        if self.consensus.votedFor > -1 {
            self.msger.Client301(msg.UID, self.consensus.votedFor)
        } else {
            self.msger.Client503(msg.UID)
        }
    case *timeout:
        if self.timer.Match(msg.version) {
            self.consensus.term += 1
            self.consensus.votedFor = self.consensus.id
            self.consensus.state = candidate
            lastIdx := len(self.consensus.log) - 1
            self.msger.BroadcastRequestVote(&RequestVote {
                self.consensus.term,
                self.consensus.id,
                self.consensus.log[lastIdx].Index,
                self.consensus.log[lastIdx].Term,
            })
            self.timer.Reset()
        }
    }
}

func (self *RaftNode) candidateHandler(m Message) {
    switch msg := m.(type) {
    case *AppendEntries:
        break
    case *RequestVote:
        break
    case *AppendEntriesResp:
        break
    case *RequestVoteResp:
        break
    case *ClientRequest:
        self.msger.Client503(msg.UID)
    case *timeout:
        if self.timer.Match(msg.version) { break }
    }
}

func (self *RaftNode) leaderHandler(m Message) {
    switch msg := m.(type) {
    case *AppendEntries:
        break
    case *RequestVote:
        break
    case *AppendEntriesResp:
        break
    case *RequestVoteResp:
        break
    case *ClientRequest:
        break
    case *timeout:
        if self.timer.Match(msg.version) { break }
    }
}

type LogEntry interface {}

type RaftLogEntry struct {
    Term uint64
    Index uint64
    Entry LogEntry
}

type Message interface {}
// either of the 6 structs below

type AppendEntries struct {
    Term uint64
    LeaderId int
    PrevLogIdx uint64
    PrevLogTerm uint64
    Entries []LogEntry
    CommitIdx uint64
}

type RequestVote struct {
    Term uint64
    CandidId int
    LastLogIdx uint64
    LastLogTerm uint64
}

type AppendEntriesResp struct {
    Term uint64
    Success bool
}

type RequestVoteResp struct {
    Term uint64
    Granted bool
}

type ClientRequest struct {
    UID uint64
    Entry LogEntry
}

type timeout struct {
    version uint64
}

// TODO configuration change request

type Messenger interface {
    // Must maintain a map from serverIds to (network) address/socket
    Register(notifch chan<- Message)
    SendAppendEntries(server int, msg *AppendEntries)
    BroadcastRequestVote(msg *RequestVote)
    Client301(uid uint64, server int) // redirect to another server (possibly the leader)
    Client503(uid uint64) // service temporarily unavailable
}

type PersistentState struct {
    T uint64
    V int
}

type Persister interface {
    LogAppend(RaftLogEntry)
    LogDiscard(uint64)
    LogRead() []RaftLogEntry
    StateRead() *PersistentState // return InitState by default
    StateSave(*PersistentState)
}

//type RaftState struct {
//    LastInclIdx uint64
//    LastInclTerm uint64
//    // configuration details
//}

type Machine interface { // should be already linked with Messenger
    Apply(uid uint64, entry LogEntry) // queue and return (non-blocking)
    //TakeSnapshot(*RaftState) // should be properly serialized with Apply
    //LoadSnapshot() *RaftState
}

func main() {}
