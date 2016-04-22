package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"github.com/critiqjo/cs733/assignment4/raft"
	"github.com/critiqjo/cs733/assignment4/store"
	"regexp"
	"strconv"
)

func init() {
	gob.RegisterName("AE", new(raft.AppendEntries))
	gob.RegisterName("AP", new(raft.AppendReply))
	gob.RegisterName("CE", new(raft.ClientEntry))
	gob.RegisterName("VQ", new(raft.VoteRequest))
	gob.RegisterName("VP", new(raft.VoteReply))
	gob.RegisterName("SR", new(store.ReqRead))
	gob.RegisterName("SW", new(store.ReqWrite))
	gob.RegisterName("SC", new(store.ReqCaS))
	gob.RegisterName("SD", new(store.ReqDelete))
}

type happyWrap struct { // make gob happy! Is there an easier way?
	Smile interface{}
}

func MsgEnc(msg raft.Message) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(&happyWrap{msg})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func MsgDec(blob []byte) (raft.Message, error) {
	var happy = new(happyWrap)
	dec := gob.NewDecoder(bytes.NewBuffer(blob))
	err := dec.Decode(happy)
	if err != nil {
		return nil, err
	}
	return happy.Smile, nil
}

// Tries to parse a ClientEntry from stream
func ParseCEntry(rstream *bufio.Reader) (*raft.ClientEntry, error) {
	// FileName is assumed to have no whitespace characters including \r and \n
	line, err := ReadLineClean(rstream)
	if err != nil { // if and only if line does not end in '\n'
		return nil, err
	}

	pat := regexp.MustCompile("^(read|write|cas|delete) (0x[0-9a-f]+) ([^ ]+)(?: ([0-9]+)(?: ([0-9]+)(?: ([0-9]+))?)?)?$")
	matches := pat.FindStringSubmatch(line)

	if len(matches) < 4 {
		return nil, errors.New("Invalid format!")
	}

	cmd := matches[1]
	uid, _ := strconv.ParseUint(matches[2], 0, 64)
	file := matches[3]
	args := matches[4:]

	if cmd == "read" && len(args[0]) == 0 {
		return cEntryWrap(uid, &store.ReqRead{
			FileName: file,
		}), nil
	} else if cmd == "write" && len(args[2]) == 0 {
		size, _ := strconv.Atoi(args[0])
		var exp uint64 = 0
		if len(args[1]) > 0 {
			exp, _ = strconv.ParseUint(args[1], 0, 64)
		}
		contents, err := reqContents(rstream, size)
		if err != nil {
			return nil, err
		}
		return cEntryWrap(uid, &store.ReqWrite{
			FileName: file,
			ExpTime:  exp,
			Contents: contents,
		}), nil
	} else if cmd == "cas" {
		ver, _ := strconv.ParseUint(args[0], 0, 64)
		size, _ := strconv.Atoi(args[1])
		var exp uint64 = 0
		if len(args[2]) > 0 {
			exp, _ = strconv.ParseUint(args[2], 0, 64)
		}
		contents, err := reqContents(rstream, size)
		if err != nil {
			return nil, err
		}
		return cEntryWrap(uid, &store.ReqCaS{
			FileName: file,
			Version:  ver,
			ExpTime:  exp,
			Contents: contents,
		}), nil
	} else if cmd == "delete" && len(args[0]) == 0 {
		return cEntryWrap(uid, &store.ReqDelete{
			FileName: file,
		}), nil
	} else {
		return nil, errors.New("Invalid format!")
	}
}

func reqContents(rstream *bufio.Reader, size int) ([]byte, error) {
	contents, err := ReadExactly(rstream, size+2)
	if err != nil {
		return nil, err
	}
	crlf := contents[size:]
	if crlf[0] != '\r' || crlf[1] != '\n' {
		return nil, errors.New("Contents did not end in CRLF")
	}
	return contents[:size], nil
}

func cEntryWrap(uid uint64, req store.Request) *raft.ClientEntry {
	return &raft.ClientEntry{
		UID:  uid,
		Data: req,
	}
}

func binaryMustEnc(val interface{}, initCap int) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, initCap))
	err := binary.Write(buf, binary.BigEndian, val)
	if err != nil {
		panic("Impossible encode error!")
	}
	return buf.Bytes()
}

func binaryMustDec(blob []byte, val interface{}) {
	buf := bytes.NewBuffer(blob)
	err := binary.Read(buf, binary.BigEndian, val)
	if err != nil {
		panic("Impossible decode error!")
	}
}

func U64Enc(val uint64) []byte {
	return binaryMustEnc(val, 8)
}

func U64Dec(blob []byte) uint64 {
	val := new(uint64)
	binaryMustDec(blob, val)
	return *val
}

func LogValEnc(entry *raft.RaftEntry) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(entry)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func LogValDec(blob []byte) (*raft.RaftEntry, error) {
	re := new(raft.RaftEntry)
	dec := gob.NewDecoder(bytes.NewBuffer(blob))
	err := dec.Decode(re)
	if err != nil {
		return nil, err
	}
	return re, nil
}

func FieldsEnc(fields *raft.RaftFields) []byte {
	return binaryMustEnc(fields, 12)
}

func FieldsDec(blob []byte) *raft.RaftFields {
	fields := new(raft.RaftFields)
	binaryMustDec(blob, fields)
	return fields
}
