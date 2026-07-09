package structs

import "github.com/cookiengineer/gows"
import "encoding/json"
import "strings"
import "sync"

type Room struct {
	Name    string                   `json:"name"`
	Clients map[*gows.WebSocket]bool `json:"clients"`
	History []Message                `json:"history"`
	mutex   sync.Mutex
}

func NewRoom(name string) *Room {

	return &Room{
		Name:    strings.ToLower(name),
		Clients: make(map[*gows.WebSocket]bool),
		History: make([]Message, 0),
		mutex:   sync.Mutex{},
	}

}

func (room *Room) AmountOfClients() int {

	room.mutex.Lock()
	defer room.mutex.Unlock()

	return len(room.Clients)

}

func (room *Room) Join(client *gows.WebSocket) {

	room.mutex.Lock()

	room.Clients[client] = true

	room.mutex.Unlock()

}

func (room *Room) Leave(client *gows.WebSocket) {

	room.mutex.Lock()

	delete(room.Clients, client)

	room.mutex.Unlock()

}

func (room *Room) Broadcast(sender *gows.WebSocket, message Message) {

	payload, err0 := json.Marshal(message)

	if err0 == nil {

		room.mutex.Lock()
		room.History = append(room.History, message)
		room.mutex.Unlock()

		for client, _ := range room.Clients {
			client.Send(payload)
		}

	}

}
