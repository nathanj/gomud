package main

import (
	"fmt"
	"github.com/nathanj/gomud/color"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type ItemType int

const (
	Potion ItemType = iota
	Head
	Torso
	Weapon
)

type Item struct {
	Name     string
	Keywords []string
	Type     ItemType
}

type Inventory struct {
	Items  []*Item
	Head   *Item
	Torso  *Item
	Weapon *Item
}

type Client struct {
	Name      string
	Conn      net.Conn
	Incoming  chan string
	Quit      chan bool
	Health    int
	MaxHealth int
	Mana      int
	MaxMana   int
	Room      *Room
	Fighting  *Enemy
	Inventory *Inventory
}

type Enemy struct {
	Name      string
	Health    int
	MaxHealth int
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
	log.Printf("Client.Close: Closing client=%p", c)
	c.Conn.Close()
	clientList = removeClient(clientList, c)
	c.Quit <- true
}

func (c *Client) handleCmdSay(args string) {
	c.Incoming <- "You say \"" + args + "\"\n"
	for _, client := range clientList {
		if client != c && client.Room == c.Room {
			client.Incoming <- c.Name + " says \"" + args + "\"\n"
		}
	}
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

func itemString(item *Item) string {
	if item == nil {
		return "None"
	}
	return item.Name
}

func (c *Client) handleCmdInventory() {
	s := "Inventory:\n"
	s += "@l@Worn on head: " + itemString(c.Inventory.Head) + "\n"
	s += "Worn on torso: " + itemString(c.Inventory.Torso) + "\n"
	s += "Wielded: " + itemString(c.Inventory.Weapon) + "@n@\n"
	for _, item := range c.Inventory.Items {
		s += item.Name + "\n"
	}
	c.Incoming <- s
}

func (c *Client) findInventoryItem(name string) *Item {
	for _, item := range c.Inventory.Items {
		for _, kw := range item.Keywords {
			if kw == name {
				return item
			}
		}
	}
	return nil
}

func (c *Client) handleCmdWear(name string) {
	item := c.findInventoryItem(name)
	if item == nil {
		c.Incoming <- "You do not have that item!\n"
		return
	}
	switch item.Type {
	case Head:
		c.Incoming <- "You equip " + item.Name + " on your head.\n"
		c.Inventory.Items = removeItem(c.Inventory.Items, item)
		if c.Inventory.Head != nil {
			c.Inventory.Items = append(c.Inventory.Items, c.Inventory.Head)
		}
		c.Inventory.Head = item
	case Torso:
		c.Incoming <- "You equip " + item.Name + " on your torso.\n"
		c.Inventory.Items = removeItem(c.Inventory.Items, item)
		if c.Inventory.Torso != nil {
			c.Inventory.Items = append(c.Inventory.Items, c.Inventory.Torso)
		}
		c.Inventory.Torso = item
	case Weapon:
		c.Incoming <- "You wield " + item.Name + " in your hand.\n"
		c.Inventory.Items = removeItem(c.Inventory.Items, item)
		if c.Inventory.Weapon != nil {
			c.Inventory.Items = append(c.Inventory.Items, c.Inventory.Weapon)
		}
		c.Inventory.Weapon = item
	default:
		c.Incoming <- "You don't know how to wear" + item.Name + "!\n"
	}
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
	case cmd == "inventory" || cmd == "inv" || cmd == "i":
		c.handleCmdInventory()
	case strings.HasPrefix(cmd, "wear "):
		c.handleCmdWear(cmd[5:])
	case strings.HasPrefix(cmd, "equip "):
		c.handleCmdWear(cmd[6:])
	case strings.HasPrefix(cmd, "eq "):
		c.handleCmdWear(cmd[3:])
	default:
		c.Incoming <- "Unknown command " + cmd + "\n"
	}
}

func ClientReader(client *Client) {
	buffer := make([]byte, 1024)

	for {
		nread, err := client.Conn.Read(buffer)
		if err != nil {
			log.Printf("ClientReader: Read: %v", err)
			break
		}
		cmd := string(buffer[0 : nread-1])
		if cmd == "quit" {
			break
		}
		log.Printf("ClientReader: %s > %s", client.Name, cmd)
		client.handleCmd(cmd)
	}
	log.Printf("ClientReader: stopped for %s", client.Name)
	client.Close()
}

func ClientSender(client *Client) {
	for {
		select {
		case buffer := <-client.Incoming:
			buf := fmt.Sprintf("%s\n%s", color.Colorize(buffer), client.makePrompt())
			count := 0
			for i := 0; i < len(buf); i++ {
				if buf[i] == 0x00 {
					break
				}
				count++
			}
			log.Printf("ClientSender: sending size=%d count=%d to %s %v\n", len(buf), count, client.Name, client.Conn.RemoteAddr())
			num, err := client.Conn.Write([]byte(buf)[0:count])
			if err != nil {
				log.Printf("ClientSender: Write: %v", err)
			} else if num != count {
				log.Printf("ClientSender: num=%d count=%d\n", num, count)
			}
		case <-client.Quit:
			log.Printf("ClientSender: quitting\n")
			return
		}
	}
}

func (r *Room) makeEnemyString() string {
	var s string
	for _, en := range r.EnemyList {
		if en.Health > 0 {
			s += fmt.Sprintf("@g@%s is here.@n@\n", en.Name)
		}
	}
	return s
}

func makeOtherPlayerString(client *Client) string {
	var s string
	for _, c := range clientList {
		if c != client && c.Room == client.Room {
			s += fmt.Sprintf("@y@%s is here.@n@\n", c.Name)
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
	c.Incoming <- c.Room.Name + "\n\n" +
		c.Room.Description + "\n" +
		c.Room.makeEnemyString() + "\n" +
		makeOtherPlayerString(c) + "\n" +
		c.Room.makeExitString() + "\n"
}

func (c *Client) makePrompt() string {
	s := ""
	if c.Fighting != nil {
		en := c.Fighting
		s = fmt.Sprintf(" Enemy %2.0f%%", 100*float32(en.Health)/float32(en.MaxHealth))
	}
	return fmt.Sprintf("%sHealth: %s%d/%d %sMana: %s%d/%d%s%s> ",
		color.NORMAL, color.B_GREEN, c.Health, c.MaxHealth,
		color.NORMAL, color.B_BLUE, c.Mana, c.MaxMana, color.NORMAL, s)
}

func handleConnection(conn net.Conn, clientChannel chan *Client) {
	log.Printf("handleConnection: Got connection from %v",
		conn.RemoteAddr())
	buffer := make([]byte, 256)
	s := fmt.Sprintf("Welcome! There are %d players connected. What is your name? ", len(clientList))
	conn.Write([]byte(s))
	nread, err := conn.Read(buffer)
	if err != nil {
		log.Printf("handleConnection: Read: %v", err)
		conn.Close()
		return
	}

	name := string(buffer[0 : nread-1])
	incoming := make(chan string)
	quit := make(chan bool)
	log.Printf("handleConnection: got name = %s", name)

	client := &Client{name, conn, incoming, quit,
		100, 100, 30, 30, room1, nil, &Inventory{nil, nil, nil, nil}}

	client.Inventory.Items = append(client.Inventory.Items,
		&Item{"Health potion", []string{"health", "potion"}, Potion})
	client.Inventory.Items = append(client.Inventory.Items,
		&Item{"Wooden Training Sword", []string{"wooden", "training", "sword"}, Weapon})
	client.Inventory.Items = append(client.Inventory.Items,
		&Item{"Leather Armor", []string{"leather", "armor"}, Torso})

	go ClientReader(client)
	go ClientSender(client)
	clientChannel <- client
	client.printRoomDescription()
}

func (c *Client) doFight() {
	en := c.Fighting
	c.Incoming <- "@g@You hit " + en.Name + " for 10 damage!@n@\n"
	en.Health -= 10
	if en.Health <= 0 {
		c.Incoming <- "@G@You kill " + en.Name + "!@n@\n"
		c.Fighting = nil
		en.Fighting = nil
		return
	}
	c.Incoming <- "@r@" + en.Name + " hits you for 5 damage!@n@\n"
	c.Health -= 5
}

func doTick() {
	for _, c := range clientList {
		if c.Fighting == nil {
			continue
		}
		c.doFight()
	}
}

func removeItem(itemList []*Item, item *Item) []*Item {
	var p []*Item
	for _, c := range itemList {
		if c != item {
			p = append(p, c)
		}
	}
	return p
}

func removeClient(clientList []*Client, client *Client) []*Client {
	var p []*Client
	for _, c := range clientList {
		if c != client {
			p = append(p, c)
		}
	}
	return p
}

func repopRoom(room *Room) {
	for _, en := range room.EnemyList {
		if en.Health <= 0 {
			log.Printf("repopRoom: repoping %v", en)
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
	log.Printf("doRepop: repop")
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

func makeEnemy(name string, health int) *Enemy {
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

	log.Printf("Listening...")

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
