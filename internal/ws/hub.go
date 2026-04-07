package ws

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type Client struct {
	Connection *websocket.Conn
	Send       chan []byte
	Channel    string
}

type Hub struct {
	clients    map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex
}

type BroadcastMessage struct {
	Channel string
	Message []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.Channel] == nil {
				h.clients[client.Channel] = make(map[*Client]bool)
			}
			h.clients[client.Channel][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.Channel]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.clients, client.Channel)
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.Lock()
			if clients, ok := h.clients[msg.Channel]; ok {
				for client := range clients {
					select {
					case client.Send <- msg.Message:
					default:
						close(client.Send)
						delete(clients, client)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) Broadcast(channel string, data any) {
	msg, err := json.Marshal(data)
	if err != nil {
		return
	}
	h.broadcast <- &BroadcastMessage{Channel: channel, Message: msg}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}
