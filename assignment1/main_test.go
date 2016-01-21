package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"testing"
	"time"
)

func init() {
	go serverMain()
	time.Sleep(100 * time.Millisecond)
}

func newTest(t *testing.T, deadline time.Duration) (net.Conn, *bufio.Reader) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Fatal(err.Error())
	}

	err = conn.SetDeadline(time.Now().Add(deadline * time.Second))
	if err != nil {
		t.Fatal(err.Error())
	}

	return conn, bufio.NewReader(conn)
}

func TestBasics(t *testing.T) {
	conn, rstream := newTest(t, 4)
	defer conn.Close()

	contents := "ab\r\ncd"

	fmt.Fprintf(conn, "write abcd %v\r\n%v\r\n",
		len(contents), contents)
	matches := expectLinePat(t, rstream, "OK ([0-9]+)\r\n")
	ver1, _ := strconv.ParseUint(matches[1], 10, 64)

	fmt.Fprintf(conn, "read abcd\r\n")
	_ = expectLinePat(t, rstream, fmt.Sprintf("CONTENTS %v %v 0 ?\r\n",
		ver1, len(contents)))
	expectContents(t, rstream, []byte(contents))

	contents = "qwer"

	fmt.Fprintf(conn, "cas abcd %v %v\r\n%v\r\n",
		ver1, len(contents), contents)
	matches = expectLinePat(t, rstream, "OK ([0-9]+)\r\n")
	ver2, _ := strconv.ParseUint(matches[1], 10, 64)

	if ver1 == ver2 {
		t.Error("Version unchanged after successive writes")
	}

	fmt.Fprintf(conn, "rea")
	time.Sleep(200 * time.Millisecond)
	fmt.Fprintf(conn, "d abcd\r\n")
	_ = expectLinePat(t, rstream, fmt.Sprintf("CONTENTS %v %v 0 ?\r\n",
		ver2, len(contents)))
	expectContents(t, rstream, []byte(contents))

	fmt.Fprintf(conn, "delete abcd\r\n")
	_ = expectLinePat(t, rstream, "OK\r\n")

	fmt.Fprintf(conn, "read abcd\r\n")
	_ = expectLinePat(t, rstream, "ERR_FILE_NOT_FOUND\r\n")
}

func TestErrors(t *testing.T) {
	conn, rstream := newTest(t, 3)
	defer conn.Close()

	fmt.Fprintf(conn, "write abcd\r\njunk\r\n")
	_ = expectLinePat(t, rstream, "ERR_CMD_ERR\r\n")

	conn, rstream = newTest(t, 3)
	defer conn.Close()

	contents := "qwer"
	fmt.Fprintf(conn, "write abcd %v\r\n%v\r\n",
		len(contents), contents)
	matches := expectLinePat(t, rstream, "OK ([0-9]+)\r\n")
	ver, _ := strconv.ParseUint(matches[1], 10, 64)

	fmt.Fprintf(conn, "cas abcd %v 4\r\njunk\r\n", ver+1)
	matches = expectLinePat(t, rstream, "ERR_VERSION ([0-9]+)\r\n")
	if ver2, _ := strconv.ParseUint(matches[1], 10, 64); ver2 != ver {
		t.Errorf("Expected version %v, found %v", ver, ver2)
	}

	fmt.Fprintf(conn, "read abcd\r\n")
	_ = expectLinePat(t, rstream, fmt.Sprintf("CONTENTS %v %v 0 ?\r\n",
		ver, len(contents)))
	expectContents(t, rstream, []byte(contents))
}

func TestTimeouts(t *testing.T) {
	conn, rstream := newTest(t, 8)
	defer conn.Close()

	contents := "\r\n"
	timeout0 := 3

	fmt.Fprintf(conn, "write file %v %v\r\n%v\r\n",
		len(contents), timeout0, contents)
	matches := expectLinePat(t, rstream, "OK ([0-9]+)\r\n")
	ver, _ := strconv.ParseUint(matches[1], 10, 64)

	fmt.Fprintf(conn, "read file\r\n")
	matches = expectLinePat(t, rstream,
		fmt.Sprintf("CONTENTS %v %v ([0-9]+) ?\r\n",
			ver, len(contents)))
	timeout1, _ := strconv.Atoi(matches[1])
	if timeout1 > timeout0 {
		t.Error("Greater timeout value:", timeout1)
	}
	expectContents(t, rstream, []byte(contents))

	if timeout1 == timeout0 {
		time.Sleep(1*time.Second + 200*time.Millisecond)
		fmt.Fprintf(conn, "read file\r\n")
		matches = expectLinePat(t, rstream,
			fmt.Sprintf("CONTENTS %v %v ([0-9]+) ?\r\n",
				ver, len(contents)))
		timeout1, _ = strconv.Atoi(matches[1])
		if timeout1 >= timeout0 {
			t.Error("Timeout did not count down:", timeout1)
		}
		expectContents(t, rstream, []byte(contents))
	}

	if timeout1 < 0 {
		t.Fatal("Negative timeout value:", timeout1)
	}

	time.Sleep(time.Duration(timeout1+1) * time.Second)
	fmt.Fprintf(conn, "read file\r\n")
	_ = expectLinePat(t, rstream, "ERR_FILE_NOT_FOUND\r\n")
}

type casResp struct {
	head string
	vers uint64
	cont string
}

func TestMultiCaS(t *testing.T) {
	conn, rstream := newTest(t, 8)
	defer conn.Close()

	clients := 16
	file := "file"
	contents := "qwer"

	fmt.Fprintf(conn, "write %v %v\r\n%v\r\n", file, len(contents), contents)
	matches := expectLinePat(t, rstream, "OK ([0-9]+)\r\n")
	ver, _ := strconv.ParseUint(matches[1], 10, 64)

	rc := make(chan casResp)
	for i := 0; i < clients; i += 1 {
		go casClient(t, rc, file, ver, contents+strconv.Itoa(i))
	}

	oked := 0
	var vernew uint64 = 0
	var contnew string

	for i := 0; i < clients; i += 1 {
		tc := time.After(1 * time.Second)
		select {
		case resp := <-rc:
			if resp.head == "OK" {
				oked += 1
				contnew = resp.cont
			}
			if i == 0 {
				vernew = resp.vers
			} else if vernew != resp.vers {
				t.Error("Response versions mismatch!")
			}
		case <-tc:
			t.Fatal("Timed out!")
		}
	}

	if vernew == ver {
		t.Error("Response version did not change!")
	} else if oked != 1 {
		t.Errorf("OK was received %v times!", oked)
	} else {
		fmt.Fprintf(conn, "read %v\r\n", file)
		_ = expectLinePat(t, rstream, fmt.Sprintf("CONTENTS %v %v 0 ?\r\n",
			vernew, len(contnew)))
		expectContents(t, rstream, []byte(contnew))
	}
}

func casClient(t *testing.T, rc chan<- casResp, file string, ver uint64, contents string) {
	conn, rstream := newTest(t, 3)
	defer conn.Close()
	fmt.Fprintf(conn, "cas %v %v %v\r\n%v\r\n", file, ver, len(contents), contents)
	matches := expectLinePat(t, rstream, "(OK|ERR_VERSION) ([0-9]+)\r\n")
	ver, _ = strconv.ParseUint(matches[2], 10, 64)
	rc <- casResp{matches[1], ver, contents}
}

func expectLinePat(t *testing.T, rstream *bufio.Reader, pattern string) []string {
	str, err := rstream.ReadString('\n')
	if err != nil {
		t.Fatal(err.Error())
	}
	reg := regexp.MustCompile(pattern)
	matches := reg.FindStringSubmatch(str)
	if len(matches) == 0 {
		t.Fatalf("Expected pattern %#v, found %#v", pattern, str)
	}
	return matches
}

func expectContents(t *testing.T, rstream *bufio.Reader, expected []byte) {
	got := getContents(t, rstream, len(expected))
	if len(expected) != len(got) {
		t.Fatal("Contents mismatch")
	} else {
		for i, b := range expected {
			if b != got[i] {
				t.Fatal("Contents mismatch")
			}
		}
	}
}

func getContents(t *testing.T, rstream *bufio.Reader, size int) []byte {
	contents := make([]byte, size)
	_, err := io.ReadFull(rstream, contents)
	if err != nil {
		t.Fatal("Bad contents:", err.Error())
	}
	trail, err := rstream.ReadSlice('\n')
	if err != nil {
		t.Fatal("Bad contents:", err.Error())
	} else if len(trail) != 2 || trail[0] != '\r' || trail[1] != '\n' {
		t.Fatal("Bad contents: did not end in CRLF")
	}
	return contents
}
