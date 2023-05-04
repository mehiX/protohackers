package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"
)

var reUsername = regexp.MustCompile(`^([0-9a-zA-Z]*[a-zA-Z]+[0-9a-zA-Z]*)$`)

type Notification struct {
	Message string
	From    string
	To      []string
}

type User struct {
	Name string
	Conn net.Conn
}

func (u User) SendMessage(msg string) error {
	_, err := fmt.Fprintln(u.Conn, msg)

	return err
}

type BudgetChat struct {
	Users map[string]User
	m     sync.RWMutex
}

func (b *BudgetChat) Usernames() []string {
	usernames := make([]string, 0)
	b.m.RLock()
	defer b.m.RUnlock()
	for username := range b.Users {
		usernames = append(usernames, username)
	}
	return usernames
}

func (b *BudgetChat) AddUser(user User) {
	// announce presence to others
	for _, u := range b.Users {
		u.SendMessage(fmt.Sprintf("* %s has entered the room", user.Name))
	}

	// show list of users to current user
	user.SendMessage(fmt.Sprintf("* The room contains: %s", strings.Join(b.Usernames(), ", ")))

	// add current user to list
	b.m.Lock()
	b.Users[user.Name] = user
	b.m.Unlock()
}

func (b *BudgetChat) DeleteUser(user User) {
	b.m.Lock()
	delete(b.Users, user.Name)
	b.m.Unlock()

	b.SendAll(fmt.Sprintf("* %s has left the room", user.Name))
}

func (b *BudgetChat) SendAll(msg string) {
	b.m.RLock()
	defer b.m.RUnlock()
	for _, u := range b.Users {
		u.SendMessage(msg)
	}
}

func (b *BudgetChat) SendAllExcept(msg string, user User) {
	b.m.RLock()
	defer b.m.RUnlock()
	for _, u := range b.Users {
		if u.Name != user.Name {
			u.SendMessage(msg)
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: budgetchat <addr>")
		os.Exit(1)
	}

	bg := &BudgetChat{
		Users: make(map[string]User, 0),
	}

	if err := bg.Start(os.Args[1]); err != nil {
		log.Fatal(err)
	}
}

func (b *BudgetChat) Start(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer l.Close()

	fmt.Printf("Listening on %s\n", addr)

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go b.manageUserSession(conn)
	}
}

func (b *BudgetChat) manageUserSession(conn net.Conn) {

	defer conn.Close()

	done := make(chan bool)
	defer close(done)

	username, err := b.askUsername(conn)
	if err != nil {
		log.Println(err)
		return
	}

	user := User{
		Name: username,
		Conn: conn,
	}

	b.AddUser(user)

	fmt.Printf("new user: %s\n", username)

	// wait for messages from user
	scnr := bufio.NewScanner(conn)
	for scnr.Scan() {
		txt := strings.TrimSpace(scnr.Text())
		if txt == "" {
			continue
		}
		fmt.Printf("%s wrote: %s\n", username, txt)
		b.SendAllExcept(fmt.Sprintf("[%s] %s", username, txt), user)
	}

	fmt.Printf("%s left the room\n", username)

	b.DeleteUser(user)

	fmt.Println("byyyyye")
}

func (b *BudgetChat) askUsername(conn net.Conn) (string, error) {
	hello := "Welcome to budgetchat! What shall I call you?"

	_, err := fmt.Fprintln(conn, hello)
	if err != nil {
		return "", err
	}

	scnr := bufio.NewScanner(conn)
	if scnr.Scan() {
		raw := scnr.Text()
		username, ok := validateUsername(raw)
		if !ok {
			//_, _ = fmt.Fprintf(conn, "* username must have between 1 and 16  alphanumeric characters\n")
			return username, fmt.Errorf("invalid username: %s", username)
		}

		return username, nil
	}

	return "", fmt.Errorf("no username provided")
}

func validateUsername(user string) (string, bool) {

	user = strings.TrimSpace(user)

	if count := utf8.RuneCountInString(user); count < 1 || count > 16 {
		return "", false
	}

	return user, reUsername.MatchString(user)
}
