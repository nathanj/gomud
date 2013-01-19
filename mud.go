package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Client struct {
	Name      string
	Conn      net.Conn
	Incoming  chan string
	Health    uint
	MaxHealth uint
	Mana      uint
	MaxMana   uint
	Room      *Room
	Fighting  *Enemy
}

type Enemy struct {
	Name      string
	Health    uint
	MaxHealth uint
	Fighting  *Client
}

func (e *Enemy) String() string {
	return fmt.Sprintf("%s [%d/%d]", e.Name, e.Health, e.MaxHealth)
}

type Room struct {
	Name           string
	Description    string
	East           *Room
	West           *Room
	North          *Room
	South          *Room
	EnemyList      []*Enemy
	EnemyListMutex sync.Mutex
}

var clientList []*Client

var room1 = &Room{Name: "Starting Room", Description: "This is the luxurious starting room."}
var room2 = &Room{Name: "Second Room", Description: "This is the second room."}

func (c *Client) Close() {
	c.Conn.Close()
}

func (c *Client) handleCmdSay(args string) {
	c.Incoming <- "You say \"" + args + "\"\n"
}

func (c *Client) findEnemy(name string) *Enemy {
	for _, en := range c.Room.EnemyList {
		if en.Health > 0 && en.Name == name {
			return en
		}
	}
	return nil
}

func (c *Client) handleCmdKill(args string) {
	if c.Fighting != nil {
		c.Incoming <- "Not while you are fighting!\n"
		return
	}
	// Lock mutex so that there is no race that allows two players
	// to begin fighting the same enemy.
	c.Room.EnemyListMutex.Lock()
	defer c.Room.EnemyListMutex.Unlock()
	en := c.findEnemy(args)
	if en == nil {
		c.Incoming <- "You do not see " + args + "\n"
		return
	}
	if en.Fighting != nil {
		c.Incoming <- args + " is already fighting!\n"
		return
	}
	c.Incoming <- "You start fighting " + args + "\n"
	c.Fighting = en
	en.Fighting = c
}

func (c *Client) handleCmdEast() {
	if c.Room.East == nil {
		c.Incoming <- "There is no exit to the east.\n"
		return
	}
	if c.Fighting != nil {
		c.Incoming <- "Not while you are fighting!\n"
		return
	}
	c.Room = c.Room.East
	c.printRoomDescription()
}

func (c *Client) handleCmdWest() {
	if c.Room.West == nil {
		c.Incoming <- "There is no exit to the west.\n"
		return
	}
	if c.Fighting != nil {
		c.Incoming <- "Not while you are fighting!\n"
		return
	}
	c.Room = c.Room.West
	c.printRoomDescription()
}

func (c *Client) handleCmdNorth() {
	if c.Room.North == nil {
		c.Incoming <- "There is no exit to the north.\n"
		return
	}
	if c.Fighting != nil {
		c.Incoming <- "Not while you are fighting!\n"
		return
	}
	c.Room = c.Room.North
	c.printRoomDescription()
}

func (c *Client) handleCmdSouth() {
	if c.Room.South == nil {
		c.Incoming <- "There is no exit to the south.\n"
		return
	}
	if c.Fighting != nil {
		c.Incoming <- "Not while you are fighting!\n"
		return
	}
	c.Room = c.Room.South
	c.printRoomDescription()
}

func (c *Client) handleCmdLook() {
	c.printRoomDescription()
}

func (c *Client) handleCmd(cmd string) {
	switch {
	case strings.HasPrefix(cmd, "say "):
		c.handleCmdSay(cmd[4:])
	case strings.HasPrefix(cmd, "\""):
		c.handleCmdSay(cmd[1:])
	case strings.HasPrefix(cmd, "'"):
		c.handleCmdSay(cmd[1:])
	case strings.HasPrefix(cmd, "kill "):
		c.handleCmdKill(cmd[5:])
	case strings.HasPrefix(cmd, "k "):
		c.handleCmdKill(cmd[2:])
	case cmd == "east" || cmd == "e":
		c.handleCmdEast()
	case cmd == "west" || cmd == "w":
		c.handleCmdWest()
	case cmd == "north" || cmd == "n":
		c.handleCmdNorth()
	case cmd == "south" || cmd == "s":
		c.handleCmdSouth()
	case cmd == "look" || cmd == "l":
		c.handleCmdLook()
	default:
		c.Incoming <- "Unknown command " + cmd + "\n"
	}
}

func ClientReader(client *Client) {
	buffer := make([]byte, 1024)

	for {
		nread, err := client.Conn.Read(buffer)
		if err != nil {
			log.Print(err)
			break
		}
		cmd := string(buffer[0 : nread-1])
		if cmd == "quit" {
			client.Close()
			break
		}
		log.Printf("ClientReader received %s> %s", client.Name, cmd)
		client.handleCmd(cmd)
	}
	log.Printf("ClientReader stopped for %s\n", client.Name)
}

func ClientSender(client *Client) {
	for {
		select {
		case buffer := <-client.Incoming:
			buf := fmt.Sprintf("%s\n%s", buffer, client.makePrompt())
			log.Print("ClientSender sending ", string(buffer), " to ", client.Name)
			count := 0
			for i := 0; i < len(buf); i++ {
				if buf[i] == 0x00 {
					break
				}
				count++
			}
			log.Print("Send size: ", count)
			client.Conn.Write([]byte(buf)[0:count])
		}
	}
}

func (r *Room) makeEnemyString() string {
	var s string
	for _, en := range r.EnemyList {
		if en.Health > 0 {
			s += fmt.Sprintf("%s is here.\n", en.Name)
		}
	}
	return s
}

func (r *Room) makeExitString() string {
	s := "You can go: "
	if r.East != nil {
		s += "east "
	}
	if r.West != nil {
		s += "west "
	}
	if r.North != nil {
		s += "north "
	}
	if r.South != nil {
		s += "south "
	}
	return s
}

func (c *Client) printRoomDescription() {
	c.Incoming <- c.Room.Name + "\n\n" + c.Room.Description + "\n" + c.Room.makeEnemyString() + "\n" + c.Room.makeExitString() + "\n"
}

func (c *Client) makePrompt() string {
	s := ""
	if c.Fighting != nil {
		en := c.Fighting
		s = fmt.Sprintf(" Enemy %2.0f%%", 100*float32(en.Health)/float32(en.MaxHealth))
	}
	return fmt.Sprintf("Health: %d/%d Mana: %d/%d%s> ",
		c.Health, c.MaxHealth, c.Mana, c.MaxMana, s)
}

func handleConnection(conn net.Conn, clientChannel chan *Client) {
	buffer := make([]byte, 256)
	conn.Write([]byte("What is your name? "))
	nread, err := conn.Read(buffer)
	if err != nil {
		log.Print("handleConnection Read() failed: ", err)
		conn.Close()
		return
	}

	name := string(buffer[0 : nread-1])
	incoming := make(chan string)
	fmt.Printf("name = %s\n", name)

	client := &Client{name, conn, incoming, 100, 100, 30, 30, room1, nil}

	go ClientReader(client)
	go ClientSender(client)
	clientChannel <- client
	client.printRoomDescription()
}

func (c *Client) doFight() {
	en := c.Fighting
	c.Incoming <- "You hit " + en.Name + " for 10 damage!\n"
	en.Health -= 10
	if en.Health <= 0 {
		c.Incoming <- "You kill " + en.Name + "!\n"
		c.Fighting = nil
		en.Fighting = nil
		return
	}
	c.Incoming <- en.Name + " hits you for 5 damage!\n"
	c.Health -= 5
}

func doTick() {
	log.Print("Doing tick")
	for _, c := range clientList {
		if c.Fighting == nil {
			continue
		}
		c.doFight()
	}
}

func repopRoom(room *Room) {
	for _, en := range room.EnemyList {
		if en.Health <= 0 {
			log.Printf("repoping %v\n", en)
			en.Health = en.MaxHealth
		}
	}
	for _, c := range clientList {
		if c.Room == room {
			c.Incoming <- "The room has repopped!"
		}
	}
}

func doRepop() {
	log.Print("Doing repop")
	repopRoom(room1)
	repopRoom(room2)
}

func Ticker(client chan *Client) {
	t := time.NewTicker(time.Second)
	repop := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-t.C:
			doTick()
		case <-repop.C:
			doRepop()
		case c := <-client:
			clientList = append(clientList, c)
		}
	}
}

func makeEnemy(name string, health uint) *Enemy {
	return &Enemy{name, health, health, nil}
}

func createWorld() {
	room1.North = room2
	room2.South = room1

	room2.EnemyList = append(room2.EnemyList,
		makeEnemy("slime", 50),
		makeEnemy("slime", 50),
		makeEnemy("horse", 50))
}

func main() {
	clientChannel := make(chan *Client)

	createWorld()

	ln, err := net.Listen("tcp", ":9998")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Listening...")

	go Ticker(clientChannel)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handleConnection(conn, clientChannel)
	}
}
