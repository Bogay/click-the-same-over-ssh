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

func (r *Room) Join(player string) int {
	r.players = append(r.players, player)
	return len(r.players) - 1
}

type RoomRepository interface {
	Create(id int) (*Room, error)
	Find(id int) *Room
	List() []*Room
	Remove(id int) error
}

type InMemoryRoomRepository struct {
	rooms map[int]*Room

	roomArr []*Room
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
	rr.updateList()
	return r, nil
}

func (rr *InMemoryRoomRepository) Find(id int) *Room {
	return rr.rooms[id]
}

func (rr *InMemoryRoomRepository) List() []*Room {
	return rr.roomArr
}

func (rr *InMemoryRoomRepository) updateList() {
	rs := make([]*Room, 0, len(rr.rooms))
	for _, r := range rr.rooms {
		rs = append(rs, r)
	}
	rr.roomArr = rs
}

func (rr *InMemoryRoomRepository) Remove(id int) error {
	if _, exists := rr.rooms[id]; exists {
		delete(rr.rooms, id)
		rr.updateList()
		return nil
	} else {
		return fmt.Errorf("id %d not exists", id)
	}
}
