package main

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
    term int
    votedFor int
    // volatile fields
    state rState
    commitIdx int
    lastAppld int
    // state-specific fields
    voteCount int // candidate
    nextIdx []int // leader
    matchIdx []int // leader
}

type RaftNode struct {
    msger Messenger
    pster Persister
    machn Machine
    notifch chan Message
    consensus rConsensus
}

func NewRaftNode(serverId int, msger Messenger, pster Persister, machn Machine) RaftNode {
    term, votedFor := pster.StateRead()
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

func (raft *RaftNode) Run() {} // event loop, waits on notifch, and for timeouts

type LogEntry interface {
    UID() uint64
}

type RaftLogEntry struct {
    Term: int
    Index: int
    Entry: LogEntry
}

type Message interface {}
// either of the 5 structs below

type AppendEntries struct {
    Term int
    LeaderId int
    PrevLogIdx int
    PrevLogTerm int
    Entries []LogEntry
    CommitIdx int
}

type RequestVote struct {
    Term int
    CandidId int
    LastLogIdx int
    LastLogTerm int
}

type AppendEntriesResp struct {
    Term int
    Success bool
}

type RequestVoteResp struct {
    Term int
    Granted bool
}

type ClientRequest struct {
    Entry LogEntry
}

type Messenger interface {
    Register(notifch chan<- Message)
    SendAppendEntries(server int, message AppendEntries)
    BroadcastRequestVote(message RequestVote)
}

type Persister interface {
    LogAppend(RaftLogEntry)
    LogCompact(int)
    LogRead() []RaftLogEntry
    StateRead() (int, int)
    StateSave(term int, votedFor int)
}

type Machine interface { // should be already linked with Messenger
    LogQueue(LogEntry) // queue for apply
    LastApplied() (int, uint64) // (# applied since last call -- ignored during the first call, uid)
}

func main() {}
