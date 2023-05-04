package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"net"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: means <addr>")
		os.Exit(1)
	}

	addr := os.Args[1]

	if err := startSrvr(addr); err != nil {
		log.Println(err)
	}
}

func startSrvr(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer l.Close()

	fmt.Printf("Listen on %s\n", addr)

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {

	defer func() func() {
		fmt.Printf("Connection from %s\n", conn.RemoteAddr())
		msg := fmt.Sprintf("Done with %s!", conn.RemoteAddr())
		return func() {
			fmt.Println(msg)
			conn.Close()
		}
	}()()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	prices := make(map[int32]int32)

	buf := make([]byte, 9)
	for {
		_, err := io.ReadFull(conn, buf)
		if err != nil {
			return
		}

		switch buf[0] {
		case 'I':
			timestamp := binary.BigEndian.Uint32(buf[1:5])
			price := binary.BigEndian.Uint32(buf[5:])
			prices[int32(timestamp)] = int32(price)
			fmt.Printf("<-- %d %d\n", timestamp, price)
		case 'Q':
			start := int32(binary.BigEndian.Uint32(buf[1:5]))
			end := int32(binary.BigEndian.Uint32(buf[5:]))

			writeMean(conn, start, end, prices)
		default:
			log.Printf("unknown command: %x\n", buf[0])
		}
	}
}

func writeMean(conn net.Conn, start, end int32, prices map[int32]int32) {
	total := big.NewInt(0)
	count := int32(0)
	if start <= end {
		for k, v := range prices {
			if k >= start && k <= end {
				total = total.Add(total, big.NewInt(int64(v)))
				count++
			}
		}
	}
	mean := float64(0)
	if count > 0 {
		mean = math.Floor(float64(total.Int64()) / float64(count))
	}
	fmt.Printf("--> %.0f\n", mean)
	conn.Write(binary.BigEndian.AppendUint32([]byte{}, uint32(mean)))
}
