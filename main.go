package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var rdb = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
	DB:   0,
})

type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	redisConn *redis.PubSub
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

var h = Hub{
	broadcast:  make(chan []byte),
	register:   make(chan *Client),
	unregister: make(chan *Client),
	clients:    make(map[*Client]bool),
}

func (h *Hub) run() {
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

func (c *Client) readPump() {
	ctx := context.Background()

	defer func() {
		h.unregister <- c
		c.redisConn.Close()
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		// Publish message to Redis
		rdb.Publish(ctx, "chat", message)
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write message to WebSocket
			c.conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	redisPubSub := rdb.Subscribe(context.Background(), "chat")
	defer redisPubSub.Close()

	client := &Client{
		conn:      conn,
		send:      make(chan []byte, 256),
		redisConn: redisPubSub,
	}
	h.register <- client
	defer func() { h.unregister <- client }()

	// Listen for incoming messages from WebSocket
	go client.readPump()

	// Send messages to WebSocket
	client.writePump()

	for {
		msg, err := redisPubSub.Receive(context.Background())
		if err != nil {
			fmt.Println(err)
			return
		}

		switch msg := msg.(type) {
		case *redis.Message:
			h.broadcast <- []byte(msg.Payload)
		}
	}
}

func main() {
	// Start WebSocket hub
	go h.run()

	// Serve WebSocket requests
	http.HandleFunc("/ws", handleConnections)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}
}
