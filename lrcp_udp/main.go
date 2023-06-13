package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: lrcp <addr>")
		os.Exit(1)
	}

	app := NewApp()

	if err := startServer(app, os.Args[1]); err != nil {
		log.Fatal(err)
	}
}

var msgConnect = regexp.MustCompile("^/connect/([0-9]+)/$")
var msgClose = regexp.MustCompile("^/close/([0-9]*)/$")
var msgAck = regexp.MustCompile("^/ack/([0-9]+)/([0-9]+)/$")
var msgData = regexp.MustCompile("^/data/([0-9]+)/([0-9]+)/([/\\a-zA-Z0-9\r\n]+)")

func startServer(app *Application, addr string) error {
	l, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer l.Close()

	buff := make([]byte, 1024)
	for {
		//fmt.Println("waiting for packets")
		n, remoteAddr, err := l.ReadFrom(buff)
		if err != nil {
			return err
		}

		if n > 1000 {
			// discard long messages
			continue
		}

		log.Printf("%s --> %s\n", remoteAddr.String(), buff[:n])

		if msgConnect.Match(buff[:n]) {
			parts := msgConnect.FindAllSubmatch(buff[:n], -1)[0]
			sessionID := string(parts[1])
			if !app.IsConnected(sessionID) {
				app.StartSession(sessionID, l, remoteAddr)
			}
			ackMsg := fmt.Sprintf("/ack/%s/%d/", sessionID, app.SessionLen(sessionID))
			send(l, ackMsg, remoteAddr)
			continue
		}

		if msgClose.Match(buff[:n]) {
			parts := msgClose.FindAllSubmatch(buff[:n], -1)[0]
			sessionID := string(parts[1])
			app.StopSession(sessionID)
			closeMsg := fmt.Sprintf("/close/%s/", sessionID)
			send(l, closeMsg, remoteAddr)
			continue
		}

		if msgData.Match(buff[:n]) {
			parts := msgData.FindAllSubmatch(buff[:n], -1)[0]
			sessionID := string(parts[1])
			pos, _ := strconv.Atoi(string(parts[2]))
			data := parts[3]
			if !bytes.HasSuffix(data, []byte("/")) {
				log.Println("doesn't end in /, discard")
				continue
			}
			data = data[:len(data)-1]
			// /data/950833135/543/illegal data/has too many/parts/
			if bytes.Count(data, []byte("/")) > bytes.Count(data, []byte(`\`)) {
				log.Println("contains unescaped slashes")
				continue
			}
			err := app.WriteTo(sessionID, pos, unescape(data))
			if err != nil && err == ErrSessionNotConnected {
				closeMsg := fmt.Sprintf("/close/%s/", sessionID)
				send(l, closeMsg, remoteAddr)
				continue
			}
			if err != nil {
				fmt.Printf("Session %s. Write error: %s\n", sessionID, err.Error())
			}

			ackMsg := fmt.Sprintf("/ack/%s/%d/", sessionID, app.SessionLen(sessionID))
			send(l, ackMsg, remoteAddr)

			continue
		}

		if msgAck.Match(buff[:n]) {
			parts := msgAck.FindAllSubmatch(buff[:n], -1)[0]
			sessionID := string(parts[1])
			length, _ := strconv.Atoi(string(parts[2]))
			if !app.IsConnected(sessionID) {
				closeMsg := fmt.Sprintf("/close/%s/", sessionID)
				send(l, closeMsg, remoteAddr)
				continue
			}

			app.AckFor(sessionID, length)

			continue
		}

		log.Println("message discarded")
	}

}

func send(l net.PacketConn, msg string, remoteAddr net.Addr) {
	log.Printf("%s <-- %s\n", remoteAddr.String(), msg)
	if _, err := l.WriteTo([]byte(msg), remoteAddr); err != nil {
		log.Println("Message not sent:", err)
	}
}
