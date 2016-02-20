# Raft implementation

## Design

This is basically a rewrite of design from the original paper.

### State

Persistent state fields: `myId`, `currentTerm`, `log[]`, `votedFor`

Volatile state fields: `commitIndex`, `lastApplied`

Volatile state fields in candidate: `voteCount`

Volatile state fields in leaders: `nextIndex[]`, `matchIndex[]`

### Log Entry

Fields: `uid`, `term`, `index`, `data`

### RPCs

#### AppendEntries

Fields: `term`, `leaderId`, `prevLogIndex`, `prevLogTerm`, `entries[]`, `commitIndex`

Response fields: `term`, `success`, `nodeId`, `modLogIndex`

#### RequestVote

Fields: `term`, `candidateId`, `lastLogIndex`, `lastLogTerm`

Response fields: `term`, `granted`, `nodeId`

### State- and event-wise response algorithm

#### Func `isUpToDate(rpc, log)`:
```
return rpc.lastLogTerm > log[-1].term || (rpc.lastLogTerm == log[-1].term &&
                                          rpc.lastLogIndex >= log[-1].index)
```

#### State: `Follower`

Reset timer on each event.

* On client request

  ```
  if votedFor > -1
      // assert votedFor != myId
      response.redirect(votedFor)
  else
      respond "retry later!"
  endif
  ```

* On timeout

  ```
  reset timer
  // check for outdated timer event?
  currentTerm += 1
  votedFor = myId
  change state to Candidate
  broadcast RequestVote to all nodes
  ```

* On `AppendEntries` RPC

  ```
  if rpc.term < currentTerm
      respond { term: currentTerm, success: false }
  else
      reset timer
      if rpc.term > currentTerm
          currentTerm = rpc.term
          votedFor = rpc.leaderId // to track leaderId
      endif

      if log[rpc.prevLogIndex] exists && log[rpc.prevLogIndex].term == rpc.prevLogTerm
          log.append(rpc.entries)
          respond { term: currentTerm, success: true }
          if commitIndex < rpc.commitIndex
              commitIndex = min(rpc.commitIndex, log[-1].index)
              if lastApplied < commitIndex
                  update state machine (and lastApplied)
          endif
          // Note: the state machine (the layer below Raft concensus) is
          // assumed to correctly handle duplicate requests using uid.
      else
          respond { term: currentTerm, success: false }
      endif
  endif
  ```

* On `RequestVote` RPC

  ```
  if rpc.term < currentTerm
      respond { term: currentTerm, granted: false }
  else
      if rpc.term > currentTerm
          currentTerm = rpc.term
          votedFor = -1
      endif

      if votedFor >= 0
          respond { term: currentTerm, granted: false }
      else if not isUpToDate(rpc, log)
          respond { term: currentTerm, granted: false }
      else
          reset timer
          votedFor = rpc.candidateId
          respond { term: currentTerm, granted: true }
      endif
  endif
  ```

* Drop RPC response packets

#### State: `Candidate`

* On client request

  ```
  respond "retry later!"
  ```

* On timeout

  ```
  // check for outdated timer event?
  currentTerm += 1
  votedFor = myId
  broadcast RequestVote to all nodes
  reset timer
  ```

* On `AppendEntries` RPC

  ```
  if rpc.term < currentTerm
      respond { term: currentTerm, success: false }
  else
      votedFor = rpc.leaderId // just needs to be non-zero
      change state to Follower
      execute Follower::AppendEntries handler
  endif
  ```

* On `RequestVote` RPC

  ```
  if rpc.term <= currentTerm
      respond { term: currentTerm, granted: false }
  else
      reset timer
      change state to Follower
      currentTerm = rpc.term
      votedFor = -1

      if not isUpToDate(rpc, log)
          respond { term: currentTerm, granted: false }
      else
          votedFor = rpc.candidateId
          respond { term: currentTerm, granted: true }
      endif
  endif
  ```

* Drop `AppendEntries` response packets

* On `RequestVote` response

  ```
  reset timer
  if response.term == currentTerm && response.granted
      voteCount += 1
      // caveat: possibility of duplicate messages
      if voteCount has reached a strict majority
          change state to Leader
          matchIndex[] = [0 ...]
          nextIndex[] = [log[-1].index + 1 ...]
      endif
  else if response.term > currentTerm
      currentTerm = response.term
      votedFor = -1
      change state to Follower
  endif
  ```

#### State: `Leader`

* On client request

  ```
  if request.uid in log
      ask the state machine to respond to the client with appropriate result
  else
      append request.entry to log with current term and next index
      broadcast the entry to up-to-date nodes
  endif
  ```

* On timeout

  ```
  broadcast heartbeat such that for node with id nodeId:
      prevLogIndex = nextIndex[nodeId] - 1,
      prevLogTerm = log[nextIndex[nodeId] - 1].term
  reset timer
  ```

* On `AppendEntries` RPC

  ```
  // assert currentTerm != rpc.term
  execute Candidate::AppendEntries handler
  ```

* On `RequestVote` RPC

  ```
  execute Candidate::RequestVote handler
  ```

* On `AppendEntries` response

  ```
  // Somewhat handwave-y pseudocode ahead!

  nodeId = response.nodeId // local variable
  if response.success == true
      if a new log entry of the current term got replicated to a majority
          commitIndex = index of latest majority replicated log
          apply whatever's pending to the state machine (queue logs, non-blocking)
          // the state machine should invoke the client handler when it finishes the operation
          // client handler could use uid to identify the client connection
      endif

      if response.modLogIndex > 0
          matchIndex[nodeId] = response.modLogIndex
          if nextIndex[nodeId] <= log[-1].index
              send a few, say C, log entries to nodeId (in a single RPC)
              nextIndex[nodeId] += C
          endif
      endif
  else if response.term == currentTerm
      // log mismatch? would matchIndex be non-zero?
      // sent packet gets dropped - next heartbeat will mismatch
      if nextIndex[nodeId] > matchIndex[nodeId] + 1
          nextIndex[nodeId] -= 1
      endif
      send a heart beat with updated prevIndex and prevTerm
  else if response.term > currentTerm
      change state to Follower
      reset timer
      votedFor = -1
      currentTerm = response.term
  else
      // drop! (misbehaving node?)
  endif
  ```

* Drop `RequestVote` response packets
