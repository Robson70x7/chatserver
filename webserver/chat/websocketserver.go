package chat

import (
	"log"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type ChatServer struct {
	Clients        map[*websocket.Conn]bool
	mux            sync.Mutex
	Name           string
	broadcast      chan Message
	messageHistory []Message
	lastUpdate     time.Time
}

type Message struct {
	Mensagem string `json:"message"`
	User     string `json:"user"`
}

func NewChatServer(name string) *ChatServer {
	return &ChatServer{
		Clients:        make(map[*websocket.Conn]bool),
		broadcast:      make(chan Message),
		Name:           name,
		messageHistory: make([]Message, 0),
	}
}

func (c *ChatServer) ShowHistoryMessages(ws *websocket.Conn) {
	c.mux.Lock()
	defer c.mux.Unlock()

	for _, msg := range c.messageHistory {
		if err := websocket.JSON.Send(ws, msg); err != nil {
			log.Printf("error sending history messages: %v\n", err)
			return
		}
	}
}

func (c *ChatServer) HandleConnections(ws *websocket.Conn) {
	c.mux.Lock()
	c.Clients[ws] = true
	c.mux.Unlock()
	defer ws.Close()

	c.ShowHistoryMessages(ws)
	for {
		var msg Message
		if err := websocket.JSON.Receive(ws, &msg); err != nil {
			c.mux.Lock()
			delete(c.Clients, ws)
			c.lastUpdate = time.Now()
			c.mux.Unlock()
			break
		}
		c.messageHistory = append(c.messageHistory, msg)
		c.broadcast <- msg
	}
}

func (c *ChatServer) HandleMessages() {
	log.Printf("starting message handler for chat %s\n", c.Name)
	for {
		msg := <-c.broadcast
		c.mux.Lock()
		for client := range c.Clients {
			if err := websocket.JSON.Send(client, msg); err != nil {
				c.mux.Unlock()
				return
			}
		}
		c.mux.Unlock()
	}
}
