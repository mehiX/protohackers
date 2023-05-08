package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"sync"

	"golang.org/x/exp/slices"
)

type service struct {
	cameraFlashes   map[string][]PlateReading // plate => []cameraFlash
	flashesMutex    sync.RWMutex
	tickets         map[uint16]map[uint16][]Ticket // road => day => []Ticket
	ticketsMutex    sync.RWMutex
	dispatchers     []Dispatcher
	dispatcherMutex sync.RWMutex
	repo            repository
}

type repository struct {
	flashes      []PlateReading
	flashesMutex sync.RWMutex
	tickets      []Ticket
	ticketsMutex sync.RWMutex
}

func SpeedDaemon() *service {
	return &service{
		cameraFlashes: make(map[string][]PlateReading),
		tickets:       make(map[uint16]map[uint16][]Ticket),
	}
}

const tolerance = 50

func (s *service) Flash(p PlateReading) {
	fmt.Printf("Register flash: %v\n", p)

	s.flashesMutex.Lock()
	arr := s.cameraFlashes[p.Plate]
	arr = append(arr, p)
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].Timestamp < arr[j].Timestamp
	})
	s.cameraFlashes[p.Plate] = arr
	s.flashesMutex.Unlock()

	if pr, ok := s.cameraFlashes[p.Plate]; ok {
		fmt.Printf("Found flashes for plate: %v\n", p.Plate)
		for i := 0; i < len(pr)-1; i++ {
			// measurings are already ordered by timestamp
			// check each consecutive pair
			avgSpeed := calculateAvgSpeed(pr[i], pr[i+1])
			if avgSpeed >= p.Limit*100+tolerance {
				fmt.Printf("Average speed exceeds limit: %v >= %v + %d\n", avgSpeed, p.Limit*100, tolerance)
				// need a ticket
				s.RegisterTicket(pr[i], pr[i+1], avgSpeed)
				//return
			} else {
				fmt.Printf("Average speed within limits: %v\n", avgSpeed)
			}
		}
	}
}

func (s *service) RegisterTicket(reading1, reading2 PlateReading, speed uint16) {
	s.ticketsMutex.Lock()
	defer s.ticketsMutex.Unlock()

	first := reading1
	second := reading2
	if reading2.Timestamp < reading1.Timestamp {
		first = reading2
		second = reading1
	}
	ticket := Ticket{
		Plate:      reading1.Plate,
		Road:       first.Road,
		Mile1:      first.Mile,
		Timestamp1: first.Timestamp,
		Mile2:      second.Mile,
		Timestamp2: second.Timestamp,
		Speed:      speed,
	}

	day := dayFromTimestamp(first.Timestamp)

	if slices.Contains(s.tickets[reading1.Road][day], ticket) {
		return
	}

	if !s.SendTicket(ticket) {
		if s.tickets[reading1.Road] == nil {
			s.tickets[reading1.Road] = make(map[uint16][]Ticket)
		}
		if s.tickets[reading1.Road][day] == nil {
			s.tickets[reading1.Road][day] = make([]Ticket, 0)
		}
		s.tickets[reading1.Road][day] = append(s.tickets[reading1.Road][day], ticket)
	}
}

func (s *service) SendTickets() {
	fmt.Println("Try to send tickets")
	s.ticketsMutex.Lock()
	defer s.ticketsMutex.Unlock()

	sentTickets := make([]Ticket, 0)
	for _, ticketsByDay := range s.tickets {
		for _, tickets := range ticketsByDay {
			for _, t := range tickets {
				if s.SendTicket(t) {
					sentTickets = append(sentTickets, t)
				}
			}
		}
	}

	fmt.Printf("Count sent tickets: %d\n", len(sentTickets))
	for _, t := range sentTickets {
		day := dayFromTimestamp(t.Timestamp1)
		if idx := slices.Index(s.tickets[t.Road][day], t); idx >= 0 {
			fmt.Println("Remove sent ticket")
			s.tickets[t.Road][day] = slices.Delete(s.tickets[t.Road][day], idx, idx)
		}
	}
}

// yyyy-MM-dd => []Ticket
var sentTicketsByDay = make(map[string][]Ticket)

func (s *service) SendTicket(t Ticket) bool {

	key := fmt.Sprintf("%s-%d", t.Plate, t.Road)
	for _, oldTicket := range sentTicketsByDay[key] {
		// check if we need a new ticket
		if (t.Mile1 == oldTicket.Mile1 && t.Timestamp1 == oldTicket.Timestamp1) ||
			(oldTicket.Mile2 == t.Mile2 && oldTicket.Timestamp2 == t.Timestamp2) ||
			(oldTicket.Mile1 == t.Mile2 && oldTicket.Timestamp1 == t.Timestamp2) ||
			(oldTicket.Mile2 == t.Mile1 && oldTicket.Timestamp2 == t.Timestamp1) {
			fmt.Printf("(1) Ticket old: %v\n", oldTicket)
			fmt.Printf("(1) Ticket new: %v\n", t)
			return true
		}

		if dayFromTimestamp(oldTicket.Timestamp1) == dayFromTimestamp(t.Timestamp1) || dayFromTimestamp(oldTicket.Timestamp2) == dayFromTimestamp(t.Timestamp2) {
			fmt.Printf("(2) Ticket old: %v\n", oldTicket)
			fmt.Printf("(2) Ticket new: %v\n", t)
			return true
		}

		if dayFromTimestamp(oldTicket.Timestamp1) != dayFromTimestamp(t.Timestamp1) &&
			dayFromTimestamp(oldTicket.Timestamp2) != dayFromTimestamp(t.Timestamp2) &&
			dayFromTimestamp(oldTicket.Timestamp2) == dayFromTimestamp(t.Timestamp1) {
			// different days, send ticket
			continue
		}

		if dayFromTimestamp(oldTicket.Timestamp1) == dayFromTimestamp(oldTicket.Timestamp2) &&
			dayFromTimestamp(t.Timestamp1) == dayFromTimestamp(t.Timestamp2) &&
			dayFromTimestamp(oldTicket.Timestamp1) != dayFromTimestamp(t.Timestamp1) {
			// different days, send ticket
			continue
		}

		if oldTicket.Mile1 >= t.Mile1 && oldTicket.Mile2 <= t.Mile2 {
			fmt.Printf("(3) Ticket old: %v\n", oldTicket)
			fmt.Printf("(3) Ticket new: %v\n", t)
			return true
		}
	}

	fmt.Printf("SendTicket: %v\n", t)
	// find a dispatcher and try to send the ticket
	s.dispatcherMutex.RLock()
	defer s.dispatcherMutex.RUnlock()

	for _, d := range s.dispatchers {
		fmt.Printf("Check dispatcher: %v\n", d)
		for _, r := range d.Roads {
			fmt.Printf("Check road: %v\n", r)
			if r == t.Road {
				if _, err := d.Conn.Write(t.Bytes()); err != nil {
					log.Printf("sending ticket: %v\n", err)
					// continue and try to find another dispatcher
					break
				} else {
					fmt.Printf("Ticket sent: %v, for date: %s\n", t, key)
					sentTicketsByDay[key] = append(sentTicketsByDay[key], t)
					return true
				}
			}
		}
	}

	fmt.Printf("Ticket not sent: %v\n", t)
	return false
}

// TicketForRoad finds the first available ticket for this road, deletes it from the system and returns it
func (s *service) TicketForRoads(roads []uint16, day uint16) (Ticket, error) {

	s.ticketsMutex.Lock()
	defer s.ticketsMutex.Unlock()

	for _, road := range roads {
		tickets := s.tickets[road][day]
		if len(tickets) > 0 {
			ticket := tickets[0]
			s.tickets[road][day] = tickets[1:]
			return ticket, nil
		}
	}

	return Ticket{}, fmt.Errorf("no tickets available")
}

func dayFromTimestamp(t uint32) uint16 {
	return uint16(math.Floor(float64(t) / 86400))
}

func calculateAvgSpeed(reading1, reading2 PlateReading) uint16 {
	// cars travel in both directions, so one reading can have bigger timestamp but smaller miles
	distance := float64(reading1.Camera.Mile - reading2.Camera.Mile)
	if reading1.Camera.Mile < reading2.Camera.Mile {
		distance = float64(reading2.Camera.Mile - reading1.Camera.Mile)
	}

	time := float64(reading1.Timestamp - reading2.Timestamp)
	if reading1.Timestamp < reading2.Timestamp {
		time = float64(reading2.Timestamp - reading1.Timestamp)
	}

	fmt.Printf("Distance: %f, Time: %f\n", distance, time)
	return uint16(distance / time * 3600 * 100)
}

func (s *service) RegisterFlash(r PlateReading) {
	s.repo.flashesMutex.Lock()
	s.repo.flashes = append(s.repo.flashes, r)
	s.repo.flashesMutex.Unlock()
}

func (s *service) RegisterDispatcher(d Dispatcher) {
	fmt.Printf("Register dispatcher: %v\n", d)
	s.dispatcherMutex.Lock()
	s.dispatchers = append(s.dispatchers, d)
	s.dispatcherMutex.Unlock()

	// TODO try to send tickets
	s.SendTickets()
}
