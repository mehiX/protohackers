package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestReceiveCameraMessages(t *testing.T) {

	sd := SpeedDaemon()

	// <-- IAmCamera{road: 123, mile: 8, limit: 60}
	// <-- Plate{plate: "UN1X", timestamp: 45}
	rdr := bytes.NewReader([]byte{0x80, 0x00, 0x7b, 0x00, 0x08, 0x00, 0x3c, 0x20, 0x04, 0x55, 0x4e, 0x31, 0x58, 0x00, 0x00, 0x00, 0x2d})
	sd.HandleSession(bufio.NewReadWriter(bufio.NewReader(rdr), nil))

	cf, ok := sd.cameraFlashes["UN1X"]
	if !ok {
		t.Fatalf("missed camera flash. got: %v", sd.cameraFlashes)
	}

	if cf[0].Road != 123 {
		t.Fatal("wrong road")
	}
	if cf[0].Timestamp != 45 {
		t.Fatal("wrong timestamp")
	}
}

func TestDispatcher(t *testing.T) {

	sd := SpeedDaemon()

	// <-- IAmCamera{road: 123, mile: 8, limit: 60}
	// <-- Plate{plate: "UN1X", timestamp: 45}
	rdr := bytes.NewReader([]byte{0x81, 0x03, 0x00, 0x42, 0x01, 0x70, 0x13, 0x88})
	sd.HandleSession(bufio.NewReadWriter(bufio.NewReader(rdr), nil))

}

func TestCalculateAvgSpeed(t *testing.T) {
	c := Camera{Road: 123, Mile: 1174, Limit: 60}
	reading1 := PlateReading{Plate: "UN1X", Timestamp: 37219279, Camera: c}

	c = Camera{Road: 123, Mile: 10, Limit: 60}
	reading2 := PlateReading{Plate: "UN1X", Timestamp: 37261183, Camera: c}

	avgSpeed := calculateAvgSpeed(reading1, reading2)

	if avgSpeed != 10000 {
		t.Fatalf("wrong average speed. expected: %d, got: %d", 10000, avgSpeed)
	}

	avgSpeed2 := calculateAvgSpeed(reading2, reading1)
	if avgSpeed != avgSpeed2 {
		t.Fatal("Should be reversible")
	}

}

func TestTicket(t *testing.T) {

	sd := SpeedDaemon()

	// <-- IAmCamera{road: 123, mile: 8, limit: 60}
	// <-- Plate{plate: "UN1X", timestamp: 00}
	rdr1 := bytes.NewReader([]byte{0x80, 0x00, 0x7b, 0x00, 0x08, 0x00, 0x3c, 0x20, 0x04, 0x55, 0x4e, 0x31, 0x58, 0x00, 0x00, 0x00, 0x00})
	sd.HandleSession(bufio.NewReadWriter(bufio.NewReader(rdr1), nil))

	// <-- IAmCamera{road: 123, mile: 9, limit: 60}
	// <-- Plate{plate: "UN1X", timestamp: 45}
	rdr2 := bytes.NewReader([]byte{0x80, 0x00, 0x7b, 0x00, 0x09, 0x00, 0x3c, 0x20, 0x04, 0x55, 0x4e, 0x31, 0x58, 0x00, 0x00, 0x00, 0x2d})
	sd.HandleSession(bufio.NewReadWriter(bufio.NewReader(rdr2), nil))

	ticket, err := sd.TicketForRoads([]uint16{123}, dayFromTimestamp(0))
	if err != nil {
		t.Fatal(err)
	}

	if ticket.Speed != 8000 {
		t.Fatal("wrong speed for ticket")
	}
}

func TestTicketBytes(t *testing.T) {

	ticket := Ticket{
		Plate:      "UN1X",
		Road:       123,
		Mile1:      8,
		Timestamp1: 0,
		Mile2:      9,
		Timestamp2: 45,
		Speed:      8000,
	}

	expect := []byte{0x21, 0x04, 0x55, 0x4e, 0x31, 0x58, 0x00, 0x7b, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x2d, 0x1f, 0x40}
	got := ticket.Bytes()

	if len(expect) != len(got) {
		t.Fatalf("wrong length. expected: %d, got: %d", len(expect), len(got))
	}

	for i := range expect {
		if expect[i] != got[i] {
			t.Fatalf("wrong byte at position %d. expected: %x, got: %x", i, expect[i], got[i])
		}
	}
}

func TestDecisecond(t *testing.T) {

	d := time.Duration(25*100) * time.Millisecond

	got := fmt.Sprintf("%v", d)
	expect := "2.5s"

	if got != expect {
		t.Fatalf("wrong decisecond conversion. expected: %s, got: %s", expect, got)
	}
}

func TestServerHeartbeat(t *testing.T) {

	addr := "127.0.0.1:45667"

	sd := SpeedDaemon()
	go sd.Start(addr)

	fmt.Println("wait for server to start")
	time.Sleep(time.Second)

	// client
	fmt.Println("connecting to server")
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		fmt.Println("send heartbeat request")
		c.Write([]byte{0x40, 0x00, 0x00, 0x00, 0x1a})
	}()

	scnr := bufio.NewScanner(c)
	scnr.Split(bufio.ScanBytes)
	count := 0

	for count < 5 && scnr.Scan() {
		fmt.Printf("Received: %v\n", scnr.Bytes())
		count++
	}
}

func TestDayFromTimestamp(t *testing.T) {

	times := []uint32{30962323, 30980949, 31024933, 31031485}

	for _, t := range times {
		fmt.Printf("%d -> %d\n", t, dayFromTimestamp(t))
	}
}
