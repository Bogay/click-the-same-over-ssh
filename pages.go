package main

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

type RoomListItem struct {
	room *Room
}

func (it *RoomListItem) FilterValue() string {
	return strconv.Itoa(it.room.id)
}

func (it *RoomListItem) Title() string {
	return fmt.Sprintf("Room #%d", it.room.id)
}

func (it *RoomListItem) Description() string {
	return fmt.Sprintf("%d / 2 players.", len(it.room.players))
}

type RoomPage struct {
	app  *App
	user string

	repo   RoomRepository
	height int
	width  int

	rooms list.Model
}

// TODO: inject repo
func NewRoomPage(height, width int, repo RoomRepository) *RoomPage {
	rawRooms := repo.List()
	items := make([]list.Item, 0, len(rawRooms))
	for _, r := range rawRooms {
		items = append(items, &RoomListItem{room: r})
	}
	rooms := list.New(items, list.NewDefaultDelegate(), width, height)
	rooms.Title = "Rooms"

	return &RoomPage{
		repo:   repo,
		height: height,
		width:  width,
		rooms:  rooms,
	}
}

func (p *RoomPage) refreshRooms() tea.Cmd {
	rawRooms := p.repo.List()
	items := make([]list.Item, 0, len(rawRooms))
	for _, r := range rawRooms {
		items = append(items, &RoomListItem{room: r})
	}
	return p.rooms.SetItems(items)
}

func (p *RoomPage) Init() tea.Cmd {
	return nil
}

func (p *RoomPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.height = msg.Height
		p.width = msg.Width
		p.rooms.SetHeight(msg.Height)
		p.rooms.SetWidth(msg.Width)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return p, tea.Quit
		case "n":
			var room *Room
			for i := 0; i < 5; i++ {
				var err error
				room, err = p.repo.Create(rand.Intn(100))
				if err == nil {
					log.Infof("add new room: %d", room.id)
					break
				}

				err = fmt.Errorf("failed to add room: %w", err)
				log.Warn(err)
				continue
			}

			if room == nil {
				// TODO: error handling
				return p, nil
			}

			_, err := p.app.tableRepo.Create(p.user)
			if err != nil {
				log.Error(err)
				return p, nil
			}

			gm := NewGameModel()
			p.app.playerToRoom[p.user] = room
			index := room.Join(p.user)

			join := func() tea.Msg {
				return Join{user: p.user, index: index}
			}
			gotoRoute := func() tea.Msg {
				return GotoRoute{route: StaticRoute{Model: &gm}}
			}
			return p, tea.Sequence(gotoRoute, join)
		case "r":
			cmd := p.refreshRooms()
			cmds = append(cmds, cmd)
			return p, tea.Batch(cmds...)
		case " ":
			item, ok := p.rooms.SelectedItem().(*RoomListItem)
			if !ok {
				log.Info("no room selected")
				return p, nil
			}

			_, err := p.app.tableRepo.Create(p.user)
			if err != nil {
				log.Error(err)
				return p, nil
			}

			room := item.room
			gm := NewGameModel()
			p.app.playerToRoom[p.user] = room

			cmds := make([]tea.Cmd, 0)
			cmds = append(cmds, func() tea.Msg {
				return GotoRoute{route: StaticRoute{Model: &gm}}
			})

			for i, player := range room.players {
				cmds = append(cmds, func() tea.Msg {
					return Join{user: player, index: i}
				})
			}

			index := room.Join(p.user)
			cmds = append(cmds, func() tea.Msg {
				// TODO: dispatch to room through app
				p.app.Send(p.user, Join{
					user:  p.user,
					index: index,
				})
				return nil
			})

			return p, tea.Sequence(cmds...)
		}
	}

	var cmd tea.Cmd
	p.rooms, cmd = p.rooms.Update(msg)
	cmds = append(cmds, cmd)

	return p, tea.Batch(cmds...)
}

func (p *RoomPage) View() string {
	return p.rooms.View()
}
