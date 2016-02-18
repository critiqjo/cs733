package main

import "time"

// Note: Raft state machine is a single-threaded event-loop
//       All events including timeouts are received on a single channel

type RaftState int

const (
    Follower RaftState = iota
    Candidate
    Leader
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
    state RaftState
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
    timer *RaftTimer
    uidIdxMap map[uint64]uint64 // uid -> idx map for entries not yet applied
    consensus rConsensus
}

func (self *RaftNode) setTermAndVote(term uint64, vote int) {
    self.consensus.term = term
    self.consensus.votedFor = vote
    self.pster.StatusSave(RaftFields { Term: term, VotedFor: vote })
}

func (self *RaftNode) setVote(vote int) {
    self.consensus.votedFor = vote
    self.pster.StatusSave(RaftFields { Term: self.consensus.term, VotedFor: vote })
}

func (self *RaftNode) logAppend(at int, entries []RaftEntry) {
    log := self.consensus.log
    // assert log[at - 1].Index + 1 == entries[0].Index
    log = append(log[:at], entries...)
    self.pster.LogUpdate(log[at:])
    self.consensus.log = log
}

func NewRaftNode(serverId int, msger Messenger, pster Persister, machn Machine) RaftNode {
    s := pster.StatusLoad()
    var term uint64
    var votedFor int
    if s == nil {
        term = 0
        votedFor = -1
    } else {
        term = s.Term
        votedFor = s.VotedFor
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
            state: Follower,
            commitIdx: 0,
            lastAppld: 0,
            voteCount: 0,
            nextIdx: nil,
            matchIdx: nil,
        },
    }
}

func (self *RaftNode) timerReset() {
    self.timer.Reset(self.consensus.state)
}

// event loop, waits on notifch, and for timeouts
func (self *RaftNode) Run(timeoutSampler func(RaftState) time.Duration) {
    self.timer = NewRaftTimer(func(v uint64) func() {
        return func() {
            self.notifch <- &timeout { v }
        }
    }, timeoutSampler)

    self.timerReset()

    for {
        msg := <-self.notifch
        switch self.consensus.state {
        case Follower:
            self.followerHandler(msg)
        case Candidate:
            self.candidateHandler(msg)
        case Leader:
            self.leaderHandler(msg)
        }
    }
}

func (self *RaftNode) isUpToDate(r *VoteRequest) bool {
    log := self.consensus.log
    lastEntry := log[len(log) - 1]
    return r.LastLogTerm > lastEntry.Term || (r.LastLogTerm == lastEntry.Term &&
                                              r.LastLogIdx >= lastEntry.Index)
}

func (self *RaftNode) followerHandler(m Message) {
    switch msg := m.(type) {
    case *AppendEntries:
        if msg.Term < self.consensus.term {
            self.msger.Send(msg.LeaderId, &AppendReply { self.consensus.term, false })
        } else {
            if msg.Term > self.consensus.term {
                self.setTermAndVote(msg.Term, msg.LeaderId) // to track leaderId
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
                        clientEntries := make([]ClientEntry, to - from)
                        for idx := from; idx < to; idx += 1 {
                            cEntry := log[idx - firstIdx].Entry
                            if cEntry != nil {
                                clientEntries[idx - from] = *cEntry
                            }
                        }
                        self.machn.ApplyLazy(clientEntries)
                        self.consensus.lastAppld = self.consensus.commitIdx
                    }
                } // else don't panic!
            } else {
                self.msger.Send(msg.LeaderId, &AppendReply { self.consensus.term, false })
            }
            self.timerReset()
        }

    case *VoteRequest:
        if msg.Term < self.consensus.term {
            self.msger.Send(msg.CandidId, &VoteReply { self.consensus.term, false })
        } else {
            if msg.Term > self.consensus.term {
                self.setTermAndVote(msg.Term, -1)
            }

            if self.consensus.votedFor >= 0 {
                self.msger.Send(msg.CandidId, &VoteReply { self.consensus.term, false })
            } else if !self.isUpToDate(msg) {
                self.msger.Send(msg.CandidId, &VoteReply { self.consensus.term, false })
            } else {
                self.setVote(msg.CandidId)
                self.msger.Send(msg.CandidId, &VoteReply { self.consensus.term, true })
                self.timerReset()
            }
        }

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
            self.consensus.state = Candidate
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
            self.setVote(msg.LeaderId) // just needs to be non-zero
            self.consensus.state = Follower
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
            self.setTermAndVote(self.consensus.term + 1, self.consensus.id)
            lastI := len(self.consensus.log) - 1
            self.msger.BroadcastVoteRequest(&VoteRequest {
                self.consensus.term,
                self.consensus.id,
                self.consensus.log[lastI].Index,
                self.consensus.log[lastI].Term,
            })
            self.timerReset()
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

func main() {}
