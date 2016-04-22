# Distributed Versioned File Store

A distributed, versioned, in-memory file store with persistent log.

Running the server is the same as in [Assignment 3](../assignment3):
```
sh$ ./assignment4 <cluster-json> <log-file> <server-id>
```

The communication protocol is given below. Fields in header lines (in both
requests and responses) are single-space (ASCII `0x20`) separated, without
leading or trailing spaces; square brackets indicate optional fields.

### Protocol

* Read a file by name:

  ```
  read <uid> <filename>\r\n
  ```
  Response on success:
  ```
  CONTENTS <version> <size> <time2exp>\r\n<content>\r\n
  ```

* Create (or overwrite) a file:

  ```
  write <uid> <filename> <size>[ <time2exp>]\r\n<content>\r\n
  ```
  Response on success:
  ```
  OK <version>\r\n
  ```

* Delete a file:

  ```
  delete <uid> <filename>\r\n
  ```
  Response on success (if exists):
  ```
  OK\r\n
  ```

* Overwrite the contents if versions match:

  ```
  cas <uid> <filename> <version> <size>[ <time2exp>]\r\n<content>\r\n
  ```
  Response on success:
  ```
  OK <version>\r\n
  ```

#### Fields

* `<uid>`: A 64-bit `0x`-prefixed hexadecimal number which uniquely identifies
  the request. A new request with the same UID will most likely return the
  cached response.
* `<filename>`: An ASCII string without any whitespace characters `[ \r\n\t]`
* `<size>`: Size of `<content>` in number of bytes (base-10 formatted)
* `<version>`: A 64-bit integer greater than zero (base-10 formatted)
* `<time2exp>`: Number of seconds after which the file will expire (base-10
  formatted integer); a zero means the file will (should) not get expired
  (default value, if ommitted in the request)
* `<content>`: Sequence of (raw) bytes

#### Error responses

* `ERRVER <current-version>\r\n`: Version mismatch (during `cas`)
* `ERR301 <current-leader>\r\n`: Redirect request
* `ERR400 Bad request\r\n`: Bad formatting
* `ERR404 File not found\r\n`
* `ERR503 Service unavailable\r\n`
* `ERR504 Service timed out\r\n`

### Points of note

* When a file is created, a 32-bit random positive integer is used as its
  initial version.
* Versions are incremented by one on each update (do not rely on this).
* The server maintains only the latest version of a file.
* When a file is expired, the file is deleted, and the version count is lost.
* When the server receives an invalid (badly formatted) request, it writes back
  `ERR_CMD_ERR\r\n`, and closes the connection.
* For `read`, the returned `<time2exp>` is `ceil` of the time to expire in
  seconds (so that `0` is only ever returned if the file has no expiration).
* For `cas`, providing a version `0` means "only create" (file must not exist).
* The server should be immune to changes of system time in Linux systems, thanks
  to [`CLOCK_MONOTONIC`](https://github.com/davecheney/junk/tree/master/clock).
* The server does not provide any hard upper-bounds on when an expired file
  will become inaccessible. The expiration time will not be in-sync across the
  cluster. Therefore, an expired file in the current leader might be readable
  from another machine if the planets line-up.
