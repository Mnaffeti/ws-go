package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type JuryRole string

const (
	JuryLeft   JuryRole = "LEFT"
	JuryRight  JuryRole = "RIGHT"
	JuryCenter JuryRole = "CENTER"
)

type Message struct {
	Jury JuryRole `json:"jury"`
	Vote string   `json:"vote"`
}

func (j JuryRole) IsValid() bool {
	return j == JuryLeft || j == JuryRight || j == JuryCenter
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
	log.Println("New Jury connected")

	for {
		var msg Message
		// Read JSON message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		if !msg.Jury.IsValid() {
			log.Printf("Invalid jury role: %s", msg.Jury)
			delete(clients, ws)
			break
		}
		log.Printf("Received -> Jury %s: %s", msg.Jury, msg.Vote)
		// Send to broadcast channel
		broadcast <- msg
	}
}

// Handle broadcasting messages to all clients
// Track votes from each jury
var juryVotes = make(map[JuryRole]string)

func handleMessages() {
	for {
		msg := <-broadcast
		// Record the vote for the jury
		juryVotes[msg.Jury] = msg.Vote

		// Check if all three juries have voted
		if len(juryVotes) == 3 {
			// Count votes
			voteCount := make(map[string]int)
			for _, vote := range juryVotes {
				voteCount[vote]++
			}
			// Find majority
			var majorityVote string
			var maxCount int
			for vote, count := range voteCount {
				if count > maxCount {
					majorityVote = vote
					maxCount = count
				}
			}
			log.Printf("All juries voted. Majority decision: %s", majorityVote)
			// Optionally, broadcast the decision to all clients
			decisionMsg := Message{Jury: "FINAL", Vote: majorityVote}
			for client := range clients {
				err := client.WriteJSON(decisionMsg)
				if err != nil {
					log.Printf("error: %v", err)
					client.Close()
					delete(clients, client)
				}
			}
			// Reset for next round
			juryVotes = make(map[JuryRole]string)
		} else {
			// Broadcast the individual vote as before
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
}

func main() {
	// Serve static test page from ./public
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", handleConnections)

	go handleMessages()

	fmt.Println("IPF server started on :8080 (LAN safe)")
	err := http.ListenAndServe("0.0.0.0:8080", nil) // listen on all interfaces
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
