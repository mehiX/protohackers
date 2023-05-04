package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: echosrvr <addr>")
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

	io.Copy(conn, conn)
}
