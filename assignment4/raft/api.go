package raft

type RaftState int

const (
    Follower RaftState = iota
    Candidate
    Leader
)

// Reserved node id (internally used to indicate that no vote was cast)
// If this value is found while calling NewNode(), it returns an error.
const NilNode uint32 = ^uint32(0)

type RaftEntry struct {
    Term uint64
    CEntry *ClientEntry
}

type Message interface {}

type AppendEntries struct {
    Term uint64
    LeaderId uint32
    PrevLogIdx uint64
    PrevLogTerm uint64
    Entries []RaftEntry
    CommitIdx uint64
}

type AppendReply struct {
    Term uint64
    Success bool
    NodeId uint32
    LastModIdx uint64
}

type ClientEntry struct {
    UID uint64
    Data interface{} // Note: Be careful while deserializing
}

type VoteRequest struct {
    Term uint64
    CandidId uint32
    LastLogIdx uint64
    LastLogTerm uint64
}

type VoteReply struct {
    Term uint64
    Granted bool
    NodeId uint32
}

// Must maintain a map from serverIds to (network) address/socket
type Messenger interface {
    // the channel through which Raft layer should be notified of new Messages
    Register(notifch chan<- Message)

    Send(node uint32, msg Message)
    BroadcastVoteRequest(msg *VoteRequest)

    // redirect to another node (possibly the leader)
    Client301(uid uint64, node uint32)

    // service temporarily unavailable (leader unknown)
    Client503(uid uint64)
}

// Caching of log could be done by the implementer
type Persister interface {
    Entry(idx uint64) *RaftEntry // return nil if out of bounds

    // if log is empty, return (0, nil); otherwise, return (last log index, last log entry)
    LastEntry() (uint64, *RaftEntry)

    // if (first index <= startIdx <= last index + 1) and startIdx <= endIdx,
    //      then return (slice, true) where slice is end-exclusive
    // otherwise return (nil, false)
    // note1: (endIdx > last index) is ok (i.e. return as much as available)
    // note2: (startIdx = last index + 1 <= endIdx) should return (nil, true)
    LogSlice(startIdx uint64, endIdx uint64) ([]RaftEntry, bool)

    // Append log entries (possibly after truncating the log from startIdx)
    LogUpdate(startIdx uint64, slice []RaftEntry) bool

    // Should return nil if no record
    GetFields() *RaftFields

    // Return whether it was successfully persisted
    SetFields(RaftFields) bool
}

type RaftFields struct {
    Term uint64
    VotedFor uint32
    // configuration details?
}

type Machine interface {
    // If the request with uid has been processed or queued, then return true
    // (so that the newly arrive request can be ignored, otherwise the request
    // will be replicated and applied), and respond to the client appropriately
    TryRespond(uid uint64) bool

    // Execute commands (possibly lazily), and respond to clients with results.
    // After this call returns, TryRespond should return true for all of these
    // uids regardless of whether the operation has been applied or is still in
    // the lazy queue.
    Execute([]ClientEntry)

    //TakeSnapshot(*LogState) // should be properly serialized with Execute
    //LoadSnapshot() *LogState
    //SerializeSnapshot() ByteStream?
}

//type LogState struct {
//    LastInclIdx uint64
//    LastInclTerm uint64
//    // configuration details?
//}
