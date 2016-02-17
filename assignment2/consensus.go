package main

import "time"

// Note: Raft state machine is a single-threaded event-loop
//       All events including timeouts are received on a single channel

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
    log []RaftEntry
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
    uidIdxMap map[uint64]uint64 // uid -> idx map for entries not yet applied
    consensus rConsensus
}

func (self *RaftNode) logAppend(off int, entries []ClientEntry) {
    log := self.consensus.log
    from := len(log)
    fromIdx := log[from - 1].Index + 1
    log = append(log, make([]RaftEntry, len(entries))...)
    for i, entry := range entries {
        log[from + i] = RaftEntry { fromIdx + uint64(i), self.consensus.term, &entry }
    }
    self.pster.LogAppend(log[from:])
    self.consensus.log = log
}

func NewRaftNode(serverId int, msger Messenger, pster Persister, machn Machine) RaftNode {
    rs := pster.StateRead()
    var term uint64
    var votedFor int
    if rs == nil {
        term = 0
        votedFor = -1
    } else {
        term = rs.Term
        votedFor = rs.VotedFor
    }
    notifch := make(chan Message)
    msger.Register(notifch)
    log := pster.LogRead()
    if log == nil {
        // simplification: to avoid a few checks for empty log
        log = []RaftEntry { RaftEntry { 0, 0, nil } }
    }
    return RaftNode {
        msger, pster, machn, notifch, nil,
        make(map[uint64]uint64),
        rConsensus {
            id: serverId,
            log: log,
            term: term,
            votedFor: votedFor,
            state: follower,
            commitIdx: 0,
            lastAppld: 0,
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
    }
}

func (self *RaftNode) followerHandler(m Message) {
    switch msg := m.(type) {
    case *AppendEntries:
        if msg.Term < self.consensus.term {
            self.msger.Send(msg.LeaderId, &VoteReply { self.consensus.term, false })
        } else {
            self.timer.Reset()
            if msg.Term > self.consensus.term {
                self.consensus.term = msg.Term
                self.consensus.votedFor = msg.LeaderId // to track leaderId
            }

            log := self.consensus.log
            prevIdx := msg.PrevLogIdx
            firstIdx := log[0].Index
            prevOff := int(prevIdx - firstIdx)
            // assert prevOff >= 0
            if int(prevOff) < len(log) && log[prevOff].Term == msg.PrevLogTerm {
                if len(msg.Entries) > 0 {
                    self.logAppend(prevOff + 1, msg.Entries)
                }
                self.msger.Send(msg.LeaderId, &AppendReply { self.consensus.term, true })
                if self.consensus.commitIdx < msg.CommitIdx {
                    lastIdx := firstIdx + uint64(len(log) - 1)
                    pracCommitIdx := msg.CommitIdx
                    if pracCommitIdx > lastIdx {
                        pracCommitIdx = lastIdx
                    }
                    self.consensus.commitIdx = pracCommitIdx
                    if self.consensus.lastAppld < pracCommitIdx {
                        from, to := self.consensus.lastAppld + 1, pracCommitIdx + 1
                        clientReqs := make([]ClientEntry, to - from)
                        for idx := from; idx < to; idx += 1 {
                            clientReqs[idx - from] = *log[idx - firstIdx].Entry // opt?
                        }
                        self.machn.ApplyLazy(clientReqs)
                        self.consensus.lastAppld = self.consensus.commitIdx
                    }
                } // else don't panic!
            } else {
                self.msger.Send(msg.LeaderId, &AppendReply { self.consensus.term, false })
            }
        }

    case *VoteRequest:
        break

    case *AppendReply:
        break

    case *VoteReply:
        break

    case *ClientEntry:
        if self.consensus.votedFor > -1 {
            self.msger.Client301(msg.UID, self.consensus.votedFor)
        } else {
            self.msger.Client503(msg.UID)
        }

    case *timeout:
        if self.timer.Match(msg.version) {
            self.consensus.state = candidate
            self.candidateHandler(msg)
        }
    }
}

func (self *RaftNode) candidateHandler(m Message) {
    switch msg := m.(type) {
    case *AppendEntries:
        if msg.Term < self.consensus.term {
            self.msger.Send(msg.LeaderId, &AppendReply { self.consensus.term, false })
        } else {
            self.consensus.votedFor = msg.LeaderId // just needs to be non-zero
            self.consensus.state = follower
            self.followerHandler(msg)
        }

    case *VoteRequest:
        break

    case *AppendReply:
        break

    case *VoteReply:
        break

    case *ClientEntry:
        self.msger.Client503(msg.UID)

    case *timeout:
        if self.timer.Match(msg.version) {
            self.consensus.term += 1
            self.consensus.votedFor = self.consensus.id
            lastI := len(self.consensus.log) - 1
            self.msger.BroadcastVoteRequest(&VoteRequest {
                self.consensus.term,
                self.consensus.id,
                self.consensus.log[lastI].Index,
                self.consensus.log[lastI].Term,
            })
            self.timer.Reset()
        }
    }
}

func (self *RaftNode) leaderHandler(m Message) {
    switch msg := m.(type) {
    case *AppendEntries:
        break
    case *VoteRequest:
        break
    case *AppendReply:
        break
    case *VoteReply:
        break
    case *ClientEntry:
        break
    case *timeout:
        if self.timer.Match(msg.version) { break }
    }
}

type RaftEntry struct {
    Index uint64 // redundant, but convinient -- Persister may optimize (also verify)
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
    Entries []ClientEntry
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

type timeout struct {
    version uint64
}

type Messenger interface {
    // Must maintain a map from serverIds to (network) address/socket
    Register(notifch chan<- Message)
    Send(server int, msg Message)
    BroadcastVoteRequest(msg *VoteRequest)
    Client301(uid uint64, server int) // redirect to another server (possibly the leader)
    Client503(uid uint64) // service temporarily unavailable
}

type RaftState struct {
    Term uint64
    VotedFor int
    // configuration details?
}

type Persister interface {
    LogAppend([]RaftEntry) // may need to discard entries based on the first index
    LogRead() []RaftEntry
    //LogReadTail(count int) []RaftEntry
    //LogReadSlice(begIdx uint64, endIdx uint64) []RaftEntry // end-exclusive
    StateRead() *RaftState // return InitState by default
    StateSave(*RaftState)
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

func main() {}
