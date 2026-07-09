package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

type BroadcastMsg struct {
	Room    string
	Event   string
	Payload interface{}
}

type RoomAction struct {
	client *Client
	room   string
}

type Hub struct {
	mu         sync.RWMutex
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	join       chan *RoomAction
	leave      chan *RoomAction
	broadcast  chan *BroadcastMsg
	stop       chan struct{}
	done       chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		join:       make(chan *RoomAction),
		leave:      make(chan *RoomAction),
		broadcast:  make(chan *BroadcastMsg, 256),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
}

func (h *Hub) Run() {
	defer close(h.done)
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.rooms[""] = map[*Client]bool{client: true}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			for room := range client.rooms {
				if clients, ok := h.rooms[room]; ok {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.rooms, room)
					}
				}
			}
			delete(h.rooms[""], client)
			if len(h.rooms[""]) == 0 {
				delete(h.rooms, "")
			}
			close(client.send)
			h.mu.Unlock()

		case action := <-h.join:
			h.mu.Lock()
			if h.rooms[action.room] == nil {
				h.rooms[action.room] = make(map[*Client]bool)
			}
			h.rooms[action.room][action.client] = true
			action.client.joinRoom(action.room)
			h.mu.Unlock()

		case action := <-h.leave:
			h.mu.Lock()
			if clients, ok := h.rooms[action.room]; ok {
				delete(clients, action.client)
				if len(clients) == 0 {
					delete(h.rooms, action.room)
				}
			}
			action.client.leaveRoom(action.room)
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			clients := h.rooms[msg.Room]
			h.mu.RUnlock()

			if len(clients) == 0 {
				continue
			}

			data, err := json.Marshal(map[string]interface{}{
				"event":     msg.Event,
				"data":      msg.Payload,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			if err != nil {
				log.Printf("ws marshal error: %v", err)
				continue
			}

			h.mu.RLock()
			for client := range clients {
				select {
				case client.send <- data:
				default:
					h.mu.RUnlock()
					h.unregister <- client
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()

		case <-h.stop:
			h.mu.Lock()
			allClients := h.rooms[""]
			h.mu.Unlock()

			for client := range allClients {
				h.unregister <- client
			}
			return
		}
	}
}

func (h *Hub) Stop() {
	close(h.stop)
	<-h.done
}

func (h *Hub) BroadcastToRoom(room, event string, data interface{}) {
	h.broadcast <- &BroadcastMsg{
		Room:    room,
		Event:   event,
		Payload: data,
	}
}
