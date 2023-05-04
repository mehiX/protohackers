package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

type db struct {
	r map[string]string
	m sync.RWMutex
}

func NewDb() *db {
	return &db{
		r: make(map[string]string),
	}
}

func (d *db) Put(key, value string) {
	d.m.Lock()
	defer d.m.Unlock()
	d.r[key] = value
}

func (d *db) Get(key string) (string, bool) {
	d.m.RLock()
	v, ok := d.r[key]
	d.m.RUnlock()
	return v, ok
}

var data = NewDb()

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: udpdb <addr>")
		os.Exit(1)
	}

	if err := startDb(os.Args[1]); err != nil {
		log.Fatal(err)
	}
}

func startDb(addr string) error {

	l, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer l.Close()

	fmt.Printf("Listen UDP on %s\n", l.LocalAddr())

	buf := make([]byte, 1000)
	for {
		n, remoteAddr, err := l.ReadFrom(buf)
		if err != nil {
			log.Println("connection", err)
			return err
		}

		fmt.Printf("Read %d bytes\n", n)
		resp, _ := handleRequest(buf[:n])
		if resp != nil {
			if _, err := l.WriteTo(resp, remoteAddr); err != nil {
				return err
			}
		}
	}
}

func handleRequest(req []byte) ([]byte, error) {
	if idx := bytes.IndexByte(req, '='); idx > 0 {
		return handleInsert(req[:idx], req[idx+1:])
	}

	if string(req) == "version" {
		return []byte("version=Mihai's version 1.0"), nil
	}

	return handleRetrieve(req)
}

func handleInsert(key, val []byte) ([]byte, error) {
	if string(key) == "version" {
		return nil, fmt.Errorf("cannot store version")
	}
	data.Put(string(key), string(val))

	return nil, nil
}

func handleRetrieve(key []byte) ([]byte, error) {
	val, _ := data.Get(string(key))
	return []byte(string(key) + "=" + string(val)), nil
}
