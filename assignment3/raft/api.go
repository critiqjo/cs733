package raft

type RaftState int

const (
    Follower RaftState = iota
    Candidate
    Leader
)

// TODO change node id type: uint32 (fixed size)

type RaftEntry struct {
    Term uint64
    CEntry *ClientEntry
}

type Message interface {}
// either of the 5 structs below

type AppendEntries struct {
    Term uint64
    LeaderId int
    PrevLogIdx uint64
    PrevLogTerm uint64
    Entries []RaftEntry
    CommitIdx uint64
}

type AppendReply struct {
    Term uint64
    Success bool
    NodeId int
    LastModIdx uint64
}

type ClientEntry struct {
    UID uint64
    Data interface{} // Note: Be careful while deserializing
}

type VoteRequest struct {
    Term uint64
    CandidId int
    LastLogIdx uint64
    LastLogTerm uint64
}

type VoteReply struct {
    Term uint64
    Granted bool
    NodeId int
}

// Must maintain a map from serverIds to (network) address/socket
type Messenger interface {
    Register(notifch chan<- Message)
    Send(node int, msg Message)
    BroadcastVoteRequest(msg *VoteRequest)
    Client301(uid uint64, nodeId int) // redirect to another node (possibly the leader)
    Client503(uid uint64) // service temporarily unavailable
}

type Persister interface { // caching of log could be done by the implementer
    Entry(idx uint64) *RaftEntry // return nil if out of bounds

    // if log is empty, return (0, nil); otherwise, return (last log index, last log entry)
    LastEntry() (uint64, *RaftEntry)

    // if n > 0 and startIdx is within bounds, return (slice, true) where 0 < len(slice) <= n
    // if n = 0 and startIdx-1 is within bounds, return (nil, true)
    // otherwise, return (nil, false)
    LogSlice(startIdx uint64, n int) ([]RaftEntry, bool)

    // Append log entries (possibly after truncating the log from startIdx)
    LogUpdate(startIdx uint64, slice []RaftEntry) bool

    GetFields() *RaftFields // return nil if no record
    SetFields(RaftFields) bool
}

type RaftFields struct {
    Term uint64
    VotedFor int
    // configuration details?
}

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

//type LogState struct {
//    LastInclIdx uint64
//    LastInclTerm uint64
//    // configuration details?
//}
