package main

import (
	"fmt"
	"time"

	"github.com/76creates/stickers/flexbox"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	app := NewApp()
	app.Start()
}

type keymap struct {
	up     key.Binding
	down   key.Binding
	left   key.Binding
	right  key.Binding
	choose key.Binding
}

type model struct {
	app     *App
	flexBox *flexbox.FlexBox

	tableLeft  ArithmeticTable
	tableRight ArithmeticTable

	timer         timer.Model
	timerProgress progress.Model

	keymap keymap
	help   help.Model
}

func newModel(keymap keymap) model {
	mathRows := make([][]ArithmeticBlock, 0)
	for i := 0; i < 4; i++ {
		r := make([]ArithmeticBlock, 0)
		for j := 0; j < 3; j++ {
			r = append(r, NewArithmeticBlock(Formula{
				lhs: 1,
				rhs: 1,
				op:  "+",
			}))
		}
		mathRows = append(mathRows, r)
	}

	// TODO: gen another table instead of copying
	mathRows2 := make([][]ArithmeticBlock, len(mathRows))
	for i, row := range mathRows {
		mathRows2[i] = make([]ArithmeticBlock, len(row))
		copy(mathRows2[i], row)
	}

	m := model{
		flexBox:       flexbox.New(0, 0),
		tableLeft:     newMathTable(mathRows),
		tableRight:    newMathTable(mathRows2),
		timer:         timer.NewWithInterval(gameDuration, time.Millisecond),
		timerProgress: progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage()),
		keymap:        keymap,
		help:          help.New(),
	}
	m.tableLeft.table[0][0].isHovered = true

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

func (t *model) renderTimer() string {
	prog := t.timerProgress.View()
	time := t.timer.Timeout.Seconds()

	return fmt.Sprintf("%s %.2fs", prog, time)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), m.timer.Init())
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	// TODO: end game
	case timer.TimeoutMsg:
		return m, tea.Quit

	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case tea.KeySpace.String():
			m.tableLeft.Toggle()
		case "q", "ctrl+c":
			return m, tea.Quit
		}

		switch {
		case key.Matches(msg, m.keymap.up):
			m.tableLeft.CursorUp()
		case key.Matches(msg, m.keymap.down):
			m.tableLeft.CursorDown()
		case key.Matches(msg, m.keymap.left):
			m.tableLeft.CursorLeft()
		case key.Matches(msg, m.keymap.right):
			m.tableLeft.CursorRight()
		}
	}

	return m, nil
}

func (m model) View() string {
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
	tableCell.SetContent(lipgloss.JoinVertical(
		lipgloss.Top,
		m.tableLeft.Render(),
	))
	tableCell = row1.GetCell(2)
	tableCell.SetContent(m.tableRight.Render())

	m.flexBox.GetRow(2).GetCell(0).SetContent(help)

	return m.flexBox.Render()
}
