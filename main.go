package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Message defines the structure of a jury vote
type Message struct {
	Jury int    `json:"jury"`
	Vote string `json:"vote"`
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // allow all origins
}

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan Message)           // broadcast channel

// Handle incoming WebSocket connections
func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	clients[ws] = true
	log.Println("New client connected")

	for {
		var msg Message
		// Read JSON message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		log.Printf("Received -> Jury %d: %s", msg.Jury, msg.Vote)
		// Send to broadcast channel
		broadcast <- msg
	}
}

// Handle broadcasting messages to all clients
func handleMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func main() {
	// Serve static test page from ./public
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", handleConnections)

	go handleMessages()

	fmt.Println("Server started on :8080 (LAN safe)")
	err := http.ListenAndServe("0.0.0.0:8080", nil) // listen on all interfaces
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
