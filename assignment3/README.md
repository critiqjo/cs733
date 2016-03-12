# Raft deployment

A simple distributed state machine (the state being represented by an integer)
supporting two operations:

* **Read**: Read the current state of the machine

  Request format: `read 0x100\r\n` where `0x100` is the unique identifier of
  that request. Responses to requests are cached by the state machine.

  Response format: `OK 8\r\n` where `8` is the current state.

* **Update**

  Request format: `update 0x101 10\r\n` where `0x101` is the unique identifier of
  that request, and 10 is the update value. Updates are done as follows:

  ```
  update(x) := state = x - state
  ```

  This is so that, given a set of update operations, the current state should be
  highly dependent on the order in which updates were applied.

  Response format: `OK 8\r\n` where `8` is the current (modified) state.

Error responses are of the form `ERR###\r\n`, where `###` is one of `301`,
`400`, `503`, `504`, which have similar meaning to the corresponding [HTTP/1.1
status codes](https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html); that is
redirection, bad request, service unavailable, and timed out, respectively.

## Terminology

* **Node**: An instance of the application (State Machine + Raft).
* **Cluster**: The collection of nodes among which a consensus has to be reached.

## Building and Running

```
sh$ go get && go build

sh$ ./assignment3 cluster.json /tmp/raftlog1.gkvl 1

sh$ ./assignment3 cluster.json /tmp/raftlog2.gkvl 2

sh$ ./assignment3 cluster.json /tmp/raftlog3.gkvl 3

sh$ telnet localhost 5011
read 0x101
OK 0
update 0x202 4
OK 4
update 0x123 7
OK 3
```

The first argument is the cluster configuration file, the second is the
key-value store file, and finally the node id.

## Cluster configuration

It is a map from node ids to network settings. See the [example config
file](./cluster.json). Each node listens for client connections at
`client-port`, and nodes communicates with each other through the `peer-port`.
And `host-ip` is exactly what you think it is!

## Components

* `SimpleMsger`: The layer which handles inter-node communication as well as
  node-client communication. Uses [`wtfnet`](./wtfnet.go) for networking, and
  [`coder`](./coder.go) for serialization.
* `SimplePster`: The layer which provides persistence of Raft log and state.
  Uses the simple-yet-awesome [`gkvlite`](https://github.com/steveyen/gkvlite)
  key-value store for persistence.
* `SimpleMachn`: The state machine which does the math!

Refer to the interfaces in [`raft/api.go`](./raft/api.go) to get a quick view
of how these interact with the Raft layer.
