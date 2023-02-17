package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type hub struct {
	clients    map[*client]bool
	broadcast  chan []byte
	register   chan *client
	unregister chan *client
}

func newHub() *hub {
	return &hub{
		broadcast:  make(chan []byte),
		register:   make(chan *client),
		unregister: make(chan *client),
		clients:    make(map[*client]bool),
	}
}

func (h *hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

type client struct {
	hub  *hub
	ws   *websocket.Conn
	send chan []byte
}

func (c *client) read() {
	defer func() {
		c.hub.unregister <- c
		c.ws.Close()
	}()
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		c.hub.broadcast <- message
	}
}

func (c *client) write() {
	defer func() {
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := c.ws.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				break
			}
		}
	}
}

func serveWs(hubInstance *hub, w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &client{
		hub:  hubInstance,
		ws:   ws,
		send: make(chan []byte, 256),
	}
	hubInstance.register <- client
	go client.write()
	client.read()
}
