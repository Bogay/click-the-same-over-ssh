package main

import "fmt"

type Room struct {
	id      int
	players []string
}

func (r *Room) RemovePlayer(player string) error {
	found := false
	n := len(r.players)
	for i, p := range r.players {
		if player == p {
			r.players[i] = r.players[n-1]
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("player %s not found", player)
	}

	r.players = r.players[:n-1]
	return nil
}

type RoomRepository interface {
	Create(id int) (*Room, error)
	Find(id int) *Room
	Remove(id int) error
}

type InMemoryRoomRepository struct {
	rooms map[int]*Room
}

func NewInMemoryRoomRepository() *InMemoryRoomRepository {
	return &InMemoryRoomRepository{
		rooms: make(map[int]*Room),
	}
}

func (rr *InMemoryRoomRepository) Create(id int) (*Room, error) {
	if _, exists := rr.rooms[id]; exists {
		return nil, fmt.Errorf("id %d used", id)
	}

	r := &Room{id: id, players: make([]string, 0)}
	rr.rooms[id] = r
	return r, nil
}

func (rr *InMemoryRoomRepository) Find(id int) *Room {
	return rr.rooms[id]
}

func (rr *InMemoryRoomRepository) Remove(id int) error {
	if _, exists := rr.rooms[id]; exists {
		delete(rr.rooms, id)
		return nil
	} else {
		return fmt.Errorf("id %d not exists", id)
	}
}
