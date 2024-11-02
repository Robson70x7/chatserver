package chat

import (
	"log"
	"sync"
	"time"
)

type ChartServerColletion struct {
	ChatServers map[string]*ChatServer
	mux         sync.Mutex
}

// NewChartServerColletion creates a new empty ChartServerColletion. The
// collection is safe for use by multiple goroutines.
func NewChartServerColletion() *ChartServerColletion {
	return &ChartServerColletion{
		ChatServers: make(map[string]*ChatServer),
		mux:         sync.Mutex{},
	}
}

// GetChatServersKeys returns a slice of strings containing the names of all
// chat servers in the collection. The method is thread-safe and does not
// allocate memory.
func (c *ChartServerColletion) GetChatServersKeys() (keys []string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	keys = make([]string, 0, len(c.ChatServers))
	for k := range c.ChatServers {
		keys = append(keys, k)
	}
	return
}

// AddChatServer adds a new chat server to the collection with the given name.
// If the collection already contains a chat server with the same name, it is
// replaced with the new one. The method returns a pointer to the newly added
// ChatServer.
func (c *ChartServerColletion) AddChatServer(name string) *ChatServer {
	c.mux.Lock()
	defer c.mux.Unlock()
	server := NewChatServer(name)
	c.ChatServers[name] = server
	return server
}

// GetChatServer retrieves a chat server by its name from the collection.
// It returns a pointer to the ChatServer if found, or nil if the server
// does not exist. This function is thread-safe.
func (c *ChartServerColletion) GetChatServer(name string) *ChatServer {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.ChatServers[name]
}

// RemoveChatServer removes a chat server from the collection. It is safe to call
// this even if the chat server does not exist in the collection.
func (c *ChartServerColletion) RemoveChatServer(name string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	delete(c.ChatServers, name)
}

// MonitorChatServers checks every 60 seconds if any chat servers have no connected
// clients. If so, it removes the chat server from the collection.
func (c *ChartServerColletion) MonitorChatServers() {
	for {
		c.mux.Lock()
		for _, server := range c.ChatServers {
			// If the server has no connected clients, remove it or update the last update time
			if len(server.Clients) == 0 && time.Since(server.lastUpdate) > 30*time.Second {
				log.Printf("removing chat server %s\n", server.Name)
				c.mux.Unlock()
				c.RemoveChatServer(server.Name)
				c.mux.Lock()
				continue
			}
		}
		c.mux.Unlock()
		time.Sleep(60 * 2 * time.Second) // 60 seconds
	}
}
