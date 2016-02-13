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
    return RaftNode {
        msger, pster, machn, notifch,
        rConsensus {
            id: serverId,
            log: pster.LogRead(),
            term: term,
            votedFor: votedFor,
            state: follower,
            commitIdx: 0,
            lastAppld: 0, // TODO init using machn.LastApplied()
            voteCount: 0,
            nextIdx: nil,
            matchIdx: nil,
        },
    }
}

func teller(ch chan Message, version uint64) func() {
    return func() {
        ch <- &timeout { version }
    }
}

func (raft *RaftNode) Run() { // event loop, waits on notifch, and for timeouts
    var timerVer uint64 = 0
    var timer *time.Timer = nil
    var timerReset = func(duration time.Duration) {
        if timer == nil || !timer.Reset(duration) {
            timerVer += 1
            timer = time.AfterFunc(duration, teller(raft.notifch, timerVer))
        }
    }
    randomDuration := 1 * time.Second
    timerReset(randomDuration)
    loop:
    for {
        switch msg := (<-raft.notifch).(type) {
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
            if msg.version == 1 { break loop }
        }
    }
}

type LogEntry interface {
    UID() uint64
}

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
    Entry LogEntry
}

type timeout struct {
    version uint64
}

type Messenger interface {
    // Must maintain a map from serverIds to (network) address/socket
    Register(notifch chan<- Message)
    SendAppendEntries(server int, message AppendEntries)
    BroadcastRequestVote(message RequestVote)
    RedirectTo(uid uint64, server int) // redirect a client/request to another server (possibly the leader)
}

type PersistentState struct {
    T uint64
    V int
}

type Persister interface {
    LogAppend(RaftLogEntry)
    LogCompact(uint64)
    LogRead() []RaftLogEntry
    StateRead() *PersistentState // return InitState by default
    StateSave(*PersistentState)
}

type MachineState struct {
    AppliedSince int
    LastUID uint64
}

type Machine interface { // should be already linked with Messenger
    LogQueue(LogEntry) // queue for apply
    LastApplied() *MachineState // (# applied since last call, uid)
}

func main() {}
