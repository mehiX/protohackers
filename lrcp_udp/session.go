package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"net"
	"time"
)

type Session2 struct {
	ID            string
	Conn          net.PacketConn
	Addr          net.Addr
	r             io.Reader
	w             io.Writer
	out           *bytes.Buffer
	pos           int // latest acknowledged position
	totalReceived int
	Ack           chan int
	Lines         chan []byte
}

func NewSession2(id string, conn net.PacketConn, addr net.Addr) *Session2 {
	r, w := io.Pipe()
	s := &Session2{
		ID:    id,
		Conn:  conn,
		Addr:  addr,
		r:     r,
		w:     w,
		out:   new(bytes.Buffer),
		Ack:   make(chan int),
		Lines: make(chan []byte),
	}
	go s.Run()
	go s.Send()

	return s
}

func (s *Session2) Run() {
	//fmt.Printf("Session %s reading\n", s.ID)
	scnr := bufio.NewScanner(s.r)
	for scnr.Scan() {
		line := scnr.Bytes()
		fmt.Printf("Session %s, got line: %s\n", s.ID, line)
		rev := revert(line)
		rev = append(rev, '\n')
		s.Lines <- rev
	}
	//fmt.Printf("Session %s done reading. Error: %v\n", s.ID, scnr.Err())
}

func (s *Session2) Write(pos int, b []byte) (int, error) {
	//fmt.Printf("Session %s. Read for pos %d, data: %s\n", s.ID, pos, b)

	if pos != s.totalReceived {
		return 0, fmt.Errorf("package not in order")
	}
	s.totalReceived += len(b)
	return s.w.Write(b)
}

func (s *Session2) Send() {
	//fmt.Printf("Session %s. Sending routine running\n", s.ID)
	tkr := time.NewTicker(2 * time.Second)
	for {
		select {
		case length := <-s.Ack:
			//fmt.Printf("Session %s. Got ACK for length: %d\n", s.ID, length)
			if length > s.out.Len() {
				//fmt.Printf("Session %s close because length in ACK is %d\n", s.ID, length)
				closeMsg := fmt.Sprintf("/close/%s/", s.ID)
				send(s.Conn, closeMsg, s.Addr)
				continue
			}

			s.pos = length
		case rev := <-s.Lines:
			s.out.Write(rev)
		case <-tkr.C:
			data := s.out.Bytes()[s.pos:]
			currentPos := s.pos
			for len(data) > 0 {
				//fmt.Printf("Session %s process data: %s\n", s.ID, data)
				nextNewline := bytes.IndexByte(data, '\n')
				if nextNewline == -1 {
					continue
				}

				maxLen := int(math.Min(float64(nextNewline), 975))
				line := data[:maxLen+1]

				dataMsg := fmt.Sprintf("/data/%s/%d/%s/", s.ID, currentPos, escape(line))
				send(s.Conn, dataMsg, s.Addr)

				currentPos += len(line)
				data = data[len(line):]
			}

		}
	}
}

func (s *Session2) Close() {
	s.r.(*io.PipeReader).Close()
}
