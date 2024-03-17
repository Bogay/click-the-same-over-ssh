package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/76creates/stickers/flexbox"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

type keymap struct {
	up     key.Binding
	down   key.Binding
	left   key.Binding
	right  key.Binding
	choose key.Binding
}

type AppModel struct {
	user    string
	app     *App
	flexBox *flexbox.FlexBox

	userLeft   string
	tableLeft  *ArithmeticTable
	userRight  string
	tableRight *ArithmeticTable

	timer         timer.Model
	timerProgress progress.Model

	keymap keymap
	help   help.Model
}

func NewAppModel(keymap keymap) AppModel {
	m := AppModel{
		flexBox:       flexbox.New(0, 0),
		timer:         timer.NewWithInterval(gameDuration, time.Millisecond),
		timerProgress: progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage()),
		keymap:        keymap,
		help:          help.New(),
	}

	flexRows := make([]*flexbox.Row, 0)
	styleScoreHeader := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center)
	row0 := m.flexBox.NewRow().AddCells(
		flexbox.NewCell(30, 1).SetStyle(styleScoreHeader),
	)
	flexRows = append(flexRows, row0)

	styleMathTable := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center)
	row1 := m.flexBox.NewRow().AddCells(
		flexbox.NewCell(4, 6).SetStyle(styleMathTable),
		flexbox.NewCell(2, 6).SetStyle(styleMathTable),
		flexbox.NewCell(4, 6).SetStyle(styleMathTable),
	)
	flexRows = append(flexRows, row1)

	styleBottomRow := lipgloss.NewStyle().Padding(1).AlignVertical(lipgloss.Bottom)
	row2 := m.flexBox.NewRow().SetStyle(styleBottomRow).AddCells(
		flexbox.NewCell(30, 2),
	)
	flexRows = append(flexRows, row2)

	m.flexBox.AddRows(flexRows)

	return m
}

func genTable() [][]ArithmeticBlock {
	mathRows := make([][]ArithmeticBlock, 0)
	for i := 0; i < 4; i++ {
		r := make([]ArithmeticBlock, 0)
		for j := 0; j < 3; j++ {
			r = append(r, NewArithmeticBlock(1+rand.Intn(13)))
		}
		mathRows = append(mathRows, r)
	}
	return mathRows
}

func (t *AppModel) renderTimer() string {
	prog := t.timerProgress.View()
	time := t.timer.Timeout.Seconds()

	return fmt.Sprintf("%s %.2fs", prog, time)
}

func (m AppModel) Init() tea.Cmd {
	go func() {

		for {
			if m.tableLeft == nil || m.tableRight == nil {
				log.Infof("waiting...")
				time.Sleep(time.Second * 1)
				continue
			}

			select {
			case evt := <-m.tableLeft.updateBlockFlagsCh:
				evt.user = m.user
				log.Debugf("send update block: %v", evt)
				go m.app.Send(m.user, evt)
			case evt := <-m.tableRight.updateBlockFlagsCh:
				evt.user = m.user
				log.Infof("send update block: %v", evt)
				go m.app.Send(m.user, evt)
			}
		}
	}()

	return tea.Batch(tickCmd(), m.timer.Init())
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.flexBox.SetHeight(msg.Height)
		m.flexBox.SetWidth(msg.Width)
	case tickMsg:
		p := float64(m.timer.Timeout.Microseconds()) / float64(gameDuration.Microseconds())
		cmd := m.timerProgress.SetPercent(p)
		return m, tea.Batch(tickCmd(), cmd)
	case progress.FrameMsg:
		progressModel, cmd := m.timerProgress.Update(msg)
		m.timerProgress = progressModel.(progress.Model)
		return m, cmd

	// case BlockFlags:
	// 	log.Debugf("update block %v", msg)
	// 	table := m.tableLeft
	// 	if msg.user == m.userRight {
	// 		table = m.tableRight
	// 	}
	// 	cell := &table.table[msg.row][msg.col]
	// 	cell.isHovered = msg.isHovered
	// 	cell.isSelected = msg.isSelected
	// 	return m, nil

	case Join:
		log.Infof("new user %s join %d", msg.user, msg.index)
		if msg.index == 0 {
			m.userLeft = msg.user
			m.tableLeft = m.app.tableRepo.FindByPlayer(msg.user)
		} else if msg.index == 1 {
			m.userRight = msg.user
			m.tableRight = m.app.tableRepo.FindByPlayer(msg.user)
		}
		return m, nil
	case Score:
		if msg.user == m.userLeft {
			m.tableLeft.score += msg.delta
		} else {
			m.tableRight.score += msg.delta
		}
		return m, nil

	// TODO: end game
	case timer.TimeoutMsg:
		return m, tea.Quit

	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

		table := m.tableLeft
		if m.user == m.userRight {
			table = m.tableRight
		}

		switch {
		case key.Matches(msg, m.keymap.up):
			table.CursorUp()
		case key.Matches(msg, m.keymap.down):
			table.CursorDown()
		case key.Matches(msg, m.keymap.left):
			table.CursorLeft()
		case key.Matches(msg, m.keymap.right):
			table.CursorRight()
		case key.Matches(msg, m.keymap.choose):
			s := table.Toggle()
			if s != 0 {
				m.app.Send(m.user, Score{user: m.user, delta: s})
			}
		}
	}

	return m, nil
}

func (m AppModel) View() string {
	m.flexBox.ForceRecalculate()
	row0 := m.flexBox.GetRow(0)
	headerCell := row0.GetCell(0)
	headerCell.SetContent(m.renderTimer())

	row1 := m.flexBox.GetRow(1)
	tableCell := row1.GetCell(0)
	help := m.help.FullHelpView([][]key.Binding{
		{
			m.keymap.up,
			m.keymap.down,
			m.keymap.left,
			m.keymap.right,
		},
		{m.keymap.choose},
	})
	leftContent := "[empty]"
	if m.tableLeft != nil {
		leftContent = m.tableLeft.Render()
	}
	tableCell.SetContent(lipgloss.JoinVertical(
		lipgloss.Top,
		leftContent,
	))
	tableCell = row1.GetCell(2)
	rightContent := "[empty]"
	if m.tableRight != nil {
		rightContent = m.tableRight.Render()
	}
	tableCell.SetContent(rightContent)

	// help += fmt.Sprintf("\nuser left:  %s", m.userLeft)
	// help += fmt.Sprintf("\nuser right: %s", m.userRight)

	m.flexBox.GetRow(2).GetCell(0).SetContent(help)

	return m.flexBox.Render()
}
