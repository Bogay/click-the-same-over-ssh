package main

type Room struct {
	id      int
	players []string
}

func NewRoom(id int) *Room {
	return &Room{id: id, players: make([]string, 0)}
}
