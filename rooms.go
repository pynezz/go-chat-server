package main

import (
	"log"
)

type Room struct {
	id         uint32
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
}

type Message struct {
	Message  string
	Type     string
	ClientId string
}

func newRoom() *Room {
	Room := &Room{
		id:         1, // TODO: Implement random room ID's if applicable in the future
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}

	go Room.run()

	return Room
}

func (r *Room) run() {
	for {
		select {
		case client := <-r.register:
			r.clients[client] = true
			log.Println("Hallo, ", client.id)
		case client := <-r.unregister:
			if _, ok := r.clients[client]; ok {
				delete(r.clients, client)
				close(client.send)
				log.Println("Ha det, ", client.id, "!")
			}
		case message := <-r.broadcast:
			for client := range r.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(r.clients, client)
				}
			}
		}
	}
}
