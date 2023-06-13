package main

import (
	"fmt"
	"net"
	"sync"
)

var ErrSessionNotConnected = fmt.Errorf("missing session")

type Application struct {
	Sessions map[string]*Session2
	m        sync.RWMutex
}

func NewApp() *Application {
	return &Application{Sessions: make(map[string]*Session2)}
}

func (a *Application) StartSession(sID string, l net.PacketConn, addr net.Addr) *Session2 {
	a.m.Lock()
	defer a.m.Unlock()

	if _, ok := a.Sessions[sID]; !ok {
		a.Sessions[sID] = NewSession2(sID, l, addr)
	}

	return a.Sessions[sID]
}

func (a *Application) StopSession(sID string) {
	a.m.Lock()
	defer a.m.Unlock()

	if s, ok := a.Sessions[sID]; ok {
		s.Close()
		delete(a.Sessions, sID)
	}
}

func (a *Application) IsConnected(sID string) bool {
	a.m.Lock()
	defer a.m.Unlock()

	_, ok := a.Sessions[sID]

	return ok
}
func (a *Application) WriteTo(sID string, pos int, data []byte) error {
	a.m.Lock()
	defer a.m.Unlock()

	if s, ok := a.Sessions[sID]; ok {
		_, err := s.Write(pos, data)
		return err
	}

	return ErrSessionNotConnected
}

func (a *Application) AckFor(id string, length int) {
	a.m.Lock()
	defer a.m.Unlock()

	if s, ok := a.Sessions[id]; ok {
		s.Ack <- length
	}
}

func (a *Application) SessionLen(sID string) int {
	a.m.Lock()
	defer a.m.Unlock()

	if s, ok := a.Sessions[sID]; ok {
		return s.totalReceived
	}

	return 0
}
