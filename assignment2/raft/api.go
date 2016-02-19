package raft

type RaftState int

const (
    Follower RaftState = iota
    Candidate
    Leader
)

// RaftNode struct { ... }
// func NewRaftNode(nodeId int, clusterSize int, Messenger, Persister, Machine) RaftNode
// func (self *RaftNode) Run(timeoutSampler func(RaftState) time.Duration)

type RaftEntry struct {
    Index uint64 // FIXME? redundant (not SPOT), but convinient
    Term uint64
    Entry *ClientEntry
}

type Message interface {}
// either of the 6 structs below

type AppendEntries struct {
    Term uint64
    LeaderId int
    PrevLogIdx uint64
    PrevLogTerm uint64
    Entries []RaftEntry
    CommitIdx uint64
}

type VoteRequest struct {
    Term uint64
    CandidId int
    LastLogIdx uint64
    LastLogTerm uint64
}

type AppendReply struct {
    Term uint64
    Success bool
}

type VoteReply struct {
    Term uint64
    Granted bool
}

type ClientEntry struct {
    UID uint64
    Data interface{}
}

type ExitLoop struct { }

type Messenger interface {
    // Must maintain a map from nodeIds to (network) address/socket
    Register(notifch chan<- Message)
    Send(node int, msg Message)
    BroadcastVoteRequest(msg *VoteRequest)
    Client301(uid uint64, nodeId int) // redirect to another node (possibly the leader)
    Client503(uid uint64) // service temporarily unavailable
}

type RaftFields struct {
    Term uint64
    VotedFor int
    // configuration details?
}

type Persister interface {
    LogUpdate([]RaftEntry) // (truncate and) append log entries
    LogRead() []RaftEntry
    //LogReadTail(count int) []RaftEntry
    //LogReadSlice(begIdx uint64, endIdx uint64) []RaftEntry // end-exclusive
    StatusLoad() *RaftFields // return InitState by default
    StatusSave(RaftFields)
}

//type LogState struct {
//    LastInclIdx uint64
//    LastInclTerm uint64
//    // configuration details?
//}

// should be internally linked with the Messenger object to respond to clients
type Machine interface {
    // if the request with uid has been processed or queued, then return true,
    // and respond to the client appropriately
    RespondIfSeen(uid uint64) bool

    // lazily apply operations; after this call, RespondIfSeen should return
    // true for all of these uids regardless of whether the operation has been
    // applied or is still in queue
    ApplyLazy([]ClientEntry)

    //TakeSnapshot(*LogState) // should be properly serialized with Apply
    //LoadSnapshot() *LogState
    //SerializeSnapshot() ByteStream?
}
