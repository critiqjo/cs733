# Versioned File Store

A simple, versioned, in-memory file store. The server listens for TCP
connections on port 8080. The communication protocol is given below (fields in
header lines (in both requests and responses) are single-space- (ASCII `0x20`)
separated, without leading or trailing spaces; square brackets indicate
optional fields).

### Protocol

* Read a file by name:

  ```
  read <filename>\r\n
  ```
  Response on success:
  ```
  CONTENTS <version> <size> <time2exp>\r\n<content>\r\n
  ```

* Create (or overwrite) a file:

  ```
  write <filename> <size>[ <time2exp>]\r\n<content>\r\n
  ```
  Response on success:
  ```
  OK <version>\r\n
  ```

* Delete a file:

  ```
  delete <filename>\r\n
  ```
  Response on success (if exists):
  ```
  OK\r\n
  ```

* Overwrite the contents if versions match:

  ```
  cas <filename> <version> <size>[ <time2exp>]\r\n<content>\r\n
  ```
  Response on success:
  ```
  OK <version>\r\n
  ```

#### Fields

* `<filename>`: An ASCII string without any whitespace characters `[ \r\n\t]`
* `<size>`: Size of `<content>` in number of bytes (base-10 formatted)
* `<version>`: A 64-bit integer greater than zero (base-10 formatted)
* `<time2exp>`: Number of seconds after which the file will expire (base-10
  formatted integer); a zero means the file will (should) not get expired
  (default value, if ommitted in the request)
* `<content>`: Sequence of (raw) bytes

#### Error responses

* `ERR_VERSION <current-version>\r\n`: Version mismatch (during `cas`)
* `ERR_FILE_NOT_FOUND\r\n`: File not found
* `ERR_CMD_ERR\r\n`: Invalid request (bad formatting)

### Points of note

* When a file is created, a random non-negative 31-bit integer is used as its
  initial version.
* Versions are incremented by one on each update (do not rely on this).
* The server maintains only the latest version of a file.
* When a file is expired, the file is deleted, and the version count is lost.
* When the server receives an invalid (badly formatted) request, it writes back
  `ERR_CMD_ERR\r\n`, and closes the connection.
* For `read`, the returned `<time2exp>` is `ceil` of the time to expire in
  seconds (so that `0` is only ever returned if the file has no expiration).
* For `cas`, providing a version `0` means "only create" (file must not exist).
* The server should be immune to changes of system time, thanks to
  [`CLOCK_MONOTONIC`](https://github.com/davecheney/junk/tree/master/clock),
  but this restricts its portability to Linux systems.
