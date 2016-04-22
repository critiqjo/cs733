package main

import "bufio"
import "bytes"
import "testing"

func TestReadLineClean(t *testing.T) {
	buf := bytes.NewBuffer([]byte("line 1234\r\nline 2345\r\n"))
	rstream := bufio.NewReader(buf)
	line, err := ReadLineClean(rstream)
	if err != nil || line != "line 1234" {
		t.Fatal("Unable to read line")
	}
	line, err = ReadLineClean(rstream)
	if err != nil || line != "line 2345" {
		t.Fatal("Unable to read line")
	}
	_, err = ReadLineClean(rstream)
	if err == nil {
		t.Fatal("Bad ending!")
	}
}
