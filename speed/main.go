package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: speed <addr>")
		os.Exit(1)
	}

	var sd = SpeedDaemon()

	if err := sd.Start(os.Args[1]); err != nil {
		log.Fatal(err)
	}
}

func (s *service) Start(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go s.HandleSession(conn)
	}
}

var transitions = map[string]map[byte][2]string{
	"reading": {
		0x20: {"plateReading", "validatePlateReading"},
		0x40: {"wantHeartbeat", "validateReadHeartbeat"},
		0x80: {"IAmCamera", "validateStartCamera"},
		0x81: {"IAmDispatcher", "validateStartDispatcher"},
		'*':  {"reading", "not supported"},
	},
	"plateReading": {
		0x00: {"readTimestamp", "skipEntry"},
		'*':  {"readPlate", "setPlateLength"},
	},
	"readPlate": {
		'*': {"readPlate", "addToPlate"},
	},
	"wantHeartbeat": {
		'*': {"wantHeartbeat", "addToHBInterval"},
	},
	"IAmCamera": {
		'*': {"IAmCamera", "addToCamera"},
	},
	"IAmDispatcher": {
		0x00: {"reading", "skipEntry"},
		'*':  {"readDispatcher", "setDispatcherLength"},
	},
	"readDispatcher": {
		'*': {"readDispatcher", "addToDispatcher"},
	},
}

// next returns [newState, action]
func next(currState string, b byte) [2]string {
	if v, ok := transitions[currState][b]; ok {
		return v
	}

	return transitions[currState]['*']
}

func (sd *service) HandleSession(conn io.ReadWriter) {

	state := "reading"

	var lenToRead = 0
	var result bytes.Buffer
	var reading PlateReading
	var hb WantHeartbeat
	var dispatcher Dispatcher

	IAmCamera := false
	IAmDispatcher := false

	scnr := bufio.NewScanner(conn)
	scnr.Split(bufio.ScanBytes)
	for scnr.Scan() {
		b := scnr.Bytes()
		nextStateAction := next(state, b[0])

		state = nextStateAction[0]

		switch nextStateAction[1] {
		case "validatePlateReading":
			if IAmDispatcher {
				msg := Error{"dispatchers don't send plate readings"}
				if _, err := conn.Write(msg.Bytes()); err != nil {
					log.Printf("sending error: %v\n", err)
				}
				return
			}
			lenToRead = 0
			result.Reset()
		case "skipEntry":
			// skip this byte
		case "setPlateLength":
			lenToRead = int(b[0]) + 4 // number + timestamp
		case "addToPlate":
			result.Write(b)
			if result.Len() == lenToRead {
				data := result.Bytes()
				reading.Plate = string(data[:len(data)-4])
				reading.Timestamp = binary.BigEndian.Uint32(data[len(data)-4:])

				fmt.Printf("Reading: %#v\n", reading)
				sd.Flash(reading)

				result.Reset()
				state = "reading"
				lenToRead = 0
			}
		case "validateReadHeartbeat":
			if hb.Interval > 0 {
				// we already had a heartbeat request
				msg := Error{"more than 1 heartbeat request"}
				if _, err := conn.Write(msg.Bytes()); err != nil {
					log.Printf("sending error for multiple HB: %v\n", err)
				}
				return
			}
			result.Reset()
			lenToRead = 4
		case "addToHBInterval":
			result.Write(b)
			if result.Len() == lenToRead {
				hb.Interval = binary.BigEndian.Uint32(result.Bytes())

				if hb.Interval > 0 {
					go sendHeartbeat(conn, hb.Interval)
				}

				lenToRead = 0
				result.Reset()
				state = "reading"
			}
		case "addToCamera":
			result.Write(b)
			if result.Len() == 6 {
				data := result.Bytes()
				reading.Road = binary.BigEndian.Uint16(data[:2])
				reading.Mile = binary.BigEndian.Uint16(data[2:4])
				reading.Limit = binary.BigEndian.Uint16(data[4:])

				lenToRead = 0
				result.Reset()
				state = "reading"
			}
		case "validateStartCamera":
			if IAmCamera {
				msg := Error{"already registered as a camera"}
				if _, err := conn.Write(msg.Bytes()); err != nil {
					log.Printf("sending error: %v\n", err)
				}
				return
			}
			if IAmDispatcher {
				msg := Error{"already registered as a dispatcher"}
				if _, err := conn.Write(msg.Bytes()); err != nil {
					log.Printf("sending error: %v\n", err)
				}
				return
			}
			result.Reset()
			lenToRead = 0
			IAmCamera = true
		case "validateStartDispatcher":
			if IAmCamera {
				msg := Error{"already registered as a camera"}
				if _, err := conn.Write(msg.Bytes()); err != nil {
					log.Printf("sending error: %v\n", err)
				}
				return
			}
			if IAmDispatcher {
				msg := Error{"already registered as a dispatcher"}
				if _, err := conn.Write(msg.Bytes()); err != nil {
					log.Printf("sending error: %v\n", err)
				}
				return
			}
			result.Reset()
			lenToRead = 0
			IAmDispatcher = true
			dispatcher = Dispatcher{Conn: conn}
		case "setDispatcherLength":
			lenToRead = int(b[0]) * 2 // multiplied by the number of bytes for each road (uint16)
		case "addToDispatcher":
			result.Write(b)
			if result.Len() == lenToRead {
				data := result.Bytes()
				for i := 0; i < lenToRead; i += 2 {
					road := binary.BigEndian.Uint16(data[i : i+2])
					dispatcher.Roads = append(dispatcher.Roads, road)
				}

				fmt.Printf("Dispatcher (length: %d): %v\n", result.Len(), dispatcher)

				sd.RegisterDispatcher(dispatcher)

				lenToRead = 0
				result.Reset()
				state = "reading"
			}
		case "not supported":
			msg := Error{"command not supported"}
			if _, err := conn.Write(msg.Bytes()); err != nil {
				log.Printf("sending error: %v\n", err)
			}
			return
		default:
			fmt.Printf("unknown action: %s\n", nextStateAction[1])
		}
	}

	if err := scnr.Err(); err != nil {
		log.Println("read error", err)
	}

}

func sendHeartbeat(w io.Writer, interval uint32) {
	// TODO: close when parent closes??
	for range time.Tick(time.Duration(interval*100) * time.Millisecond) {
		fmt.Println("send heartbeat")
		if l, err := w.Write([]byte{0x41}); err != nil {
			log.Printf("sending heartbeat: %v\n", err)
			return
		} else {
			fmt.Printf("sent %d bytes\n", l)
		}
	}

}
