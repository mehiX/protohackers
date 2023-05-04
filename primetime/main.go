package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: primetime <addr>")
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

type Request struct {
	Method *string `json:"method"`
	Number *int    `json:"number"`
}

func (r Request) String() string {
	return fmt.Sprint(r.Number)
}

func (r Request) Validate() error {
	if r.Method == nil || r.Number == nil {
		return fmt.Errorf("missing value")
	}

	if *r.Method != "isPrime" {
		return fmt.Errorf("wrong method: %s", *r.Method)
	}

	return nil
}

func isPrime(n int) bool {
	i := big.NewInt(int64(n))
	return i.ProbablyPrime(1)
}

type Response struct {
	Method  string `json:"method"`
	IsPrime bool   `json:"prime"`
}

func handleConn(conn net.Conn) {

	defer conn.Close()

	scnr := bufio.NewScanner(conn)

	for scnr.Scan() {
		resp, err := process(scnr.Text())
		if err != nil {
			conn.Write([]byte(err.Error() + "\n"))
			continue
		}

		json.NewEncoder(conn).Encode(resp)
	}
}

func process(l string) (Response, error) {

	var r Request
	var resp Response

	if err := json.Unmarshal([]byte(l), &r); err != nil {
		if !strings.Contains(err.Error(), "cannot unmarshal number") {
			return resp, err
		} else {
			return Response{"isPrime", false}, r.Validate()
		}
	}

	if err := r.Validate(); err != nil {
		return resp, err
	}

	return Response{Method: "isPrime", IsPrime: isPrime(*r.Number)}, nil
}
