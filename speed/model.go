package main

import (
	"bytes"
	"encoding/binary"
	"io"
)

type PlateReading struct {
	Plate     string
	Timestamp uint32
	Camera
}

type Camera struct {
	Road  uint16
	Mile  uint16
	Limit uint16
}

type Dispatcher struct {
	Roads []uint16
	Conn  io.ReadWriter
}

type Ticket struct {
	Plate      string
	Road       uint16
	Mile1      uint16
	Timestamp1 uint32
	Mile2      uint16
	Timestamp2 uint32
	Speed      uint16
}

func (t Ticket) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(0x21)
	buf.WriteByte(byte(len(t.Plate)))
	buf.Write([]byte(t.Plate))
	data := buf.Bytes()
	data = binary.BigEndian.AppendUint16(data, t.Road)
	data = binary.BigEndian.AppendUint16(data, t.Mile1)
	data = binary.BigEndian.AppendUint32(data, t.Timestamp1)
	data = binary.BigEndian.AppendUint16(data, t.Mile2)
	data = binary.BigEndian.AppendUint32(data, t.Timestamp2)
	data = binary.BigEndian.AppendUint16(data, t.Speed)

	return data
}

type WantHeartbeat struct {
	Interval uint32
}

type Error struct {
	Message string `json:"msg"`
}

func (e Error) Bytes() []byte {
	m := []byte(e.Message)
	var buf bytes.Buffer
	buf.WriteByte(0x10)
	buf.WriteByte(byte(len(m)))
	buf.Write(m)
	return buf.Bytes()
}
