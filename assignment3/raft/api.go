package raft

type RaftState int

const (
    Follower RaftState = iota
    Candidate
    Leader
)

// reserved node id (internally used to indicate that no vote was cast)
// if this value is found while calling NewNode(), it returns an error
const NilNode uint32 = ^uint32(0)

type RaftEntry struct {
    Term uint64
    CEntry *ClientEntry
}

type Message interface {}
// either of the 5 structs below

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
    Register(notifch chan<- Message)
    Send(node uint32, msg Message)
    BroadcastVoteRequest(msg *VoteRequest)
    Client301(uid uint64, node uint32) // redirect to another node (possibly the leader)
    Client503(uid uint64) // service temporarily unavailable
}

type Persister interface { // caching of log could be done by the implementer
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

    GetFields() *RaftFields // return nil if no record
    SetFields(RaftFields) bool
}

type RaftFields struct {
    Term uint64
    VotedFor uint32
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
