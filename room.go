package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type room struct {
	//forward is a channel that holds incoming messages
	//that should be forwarded to the other clients
	forward chan []byte
	//a channel for clients wishing to join the room
	join chan *client
	//leave is channel for clients wishing to leave the room
	leave chan *client
	//cliensts holds all current clients in this room
	clients map[*client]bool
}

func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
	}
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			//joining
			r.clients[client] = true
		case client := <-r.leave:
			//leaving
			delete(r.clients, client)
			close(client.send)
		case msg := <-r.forward:
			//forward message to all clients
			for client := range r.clients {
				client.send <- msg
			}

		}
	}
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  socketBufferSize,
	WriteBufferSize: socketBufferSize,
}

//Turning a room into an HTTP handler
func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	//getting the socket
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServerHTTP: ", err)
		return
	}
	//Create the client
	client := &client{
		socket: socket,
		send:   make(chan []byte, messageBufferSize),
		room:   r,
	}
	//pass the newly created client into the join channel for the current room
	r.join <- client
	//leaving operation when the client is finished
	defer func() { r.leave <- client }()

	go client.write()
	//Calling the read method in the main thread which will block operations (keeping the connection alive)
	client.read()
}
