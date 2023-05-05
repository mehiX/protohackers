package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: proxy <addr> <remote:port>")
		os.Exit(1)
	}

	if err := startProxy(os.Args[1], os.Args[2]); err != nil {
		log.Fatal(err)
	}
}

func startProxy(localAddr, remoteAddr string) error {

	// connect local
	l, err := net.Listen("tcp", localAddr)
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		lc, err := l.Accept()
		if err != nil {
			return err
		}

		// bind
		go handleConn(lc, remoteAddr)
	}

}

func handleConn(local net.Conn, remoteAddr string) {
	defer local.Close()

	// connect remote
	rc, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		log.Println("no conn to remote", err)
		return
	}
	defer rc.Close()

	go func() {
		defer local.Close()
		hijack(local, rc)
	}()

	hijack(rc, local)
}

var pattern = regexp.MustCompile(`^7[0-9a-zA-Z]{25,34}$`)

const (
	coinAddr = `7YWHMfk9JZe0LM0g1ZauHuiSxhI`
)

func hijack(dest, src io.ReadWriter) {

	dropCR := func(data []byte) []byte {
		if len(data) > 0 && data[len(data)-1] == '\r' {
			return data[0 : len(data)-1]
		}
		return data
	}
	// needed to pass test 4. If we receive data that does not end in `\n` then we should not forward that to the server
	// Copied from the bufio package
	splitFullLines := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			// We have a full newline-terminated line.
			return i + 1, dropCR(data[0:i]), nil
		}
		// If we're at EOF, we have a final, non-terminated line. DO NOT Return it.
		if atEOF {
			return 0, nil, nil
		}
		// Request more data.
		return 0, nil, nil
	}

	scnr := bufio.NewScanner(src)
	scnr.Split(splitFullLines)
	for scnr.Scan() {
		orig := scnr.Text()
		txt := replaceCoinAddr(orig)

		fmt.Printf("'%s' >> '%s'\n", orig, txt)
		if _, err := fmt.Fprintln(dest, txt); err != nil {
			log.Println("remote read error", err)
			return
		}

	}
}

func replaceCoinAddr(str string) string {
	//fmt.Printf("R: '%s'\n", str)
	parts := strings.Split(str, " ")
	for i := range parts {
		//fmt.Printf("T: '%s'\n", parts[i])
		if pattern.MatchString(strings.TrimSpace(parts[i])) {
			//fmt.Printf("F: '%s'\n", parts[i])
			parts[i] = coinAddr
		}
	}

	resp := strings.Join(parts, " ")
	//fmt.Printf("'%s'\n", resp)

	return resp
}
