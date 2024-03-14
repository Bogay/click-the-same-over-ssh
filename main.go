package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/76creates/stickers/flexbox"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
)

const (
	host         = "localhost"
	port         = "23234"
	gameDuration = time.Second * 60
)

var (
	colorWhite         = lipgloss.Color("#ffffff")
	colorHovered       = lipgloss.Color("#f368e0")
	styleBlockNormal   = lipgloss.NewStyle().Foreground(colorWhite).BorderForeground(colorWhite)
	styleBlockHovered  = lipgloss.NewStyle().Foreground(colorHovered).BorderForeground(colorHovered)
	styleBlockSelected = lipgloss.NewStyle().Background(colorHovered).Foreground(colorWhite)
)

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			myCustomBubbleteaMiddleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Fatal("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func myCustomBubbleteaMiddleware() wish.Middleware {
	newProg := func(m tea.Model, opts ...tea.ProgramOption) *tea.Program {
		p := tea.NewProgram(m, opts...)
		return p
	}
	teaHandler := func(s ssh.Session) *tea.Program {
		_, _, active := s.Pty()
		if !active {
			wish.Fatalln(s, "no active terminal, skipping")
			return nil
		}

		m := newModel(keymap{
			up:     key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
			down:   key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
			left:   key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "left")),
			right:  key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "right")),
			choose: key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "(un)select")),
		})

		return newProg(m, append(bubbletea.MakeOptions(s), tea.WithAltScreen())...)
	}
	return bubbletea.MiddlewareWithProgramHandler(teaHandler, termenv.ANSI256)
}

type mathFormula struct {
	lhs int
	rhs int
	op  string
}

func (f *mathFormula) Render() string {
	styleOperand := lipgloss.NewStyle().Width(2)
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		styleOperand.Align(lipgloss.Right).Render(strconv.Itoa(f.lhs)),
		" ",
		f.op,
		" ",
		styleOperand.Align(lipgloss.Left).Render(strconv.Itoa(f.rhs)),
	)
}

type mathBlock struct {
	formula    mathFormula
	value      int
	isSelected bool
	isHovered  bool
}

func newMathBlock(formula mathFormula) mathBlock {
	return mathBlock{
		formula:    formula,
		value:      0,
		isSelected: false,
		isHovered:  false,
	}
}

func (b *mathBlock) Render() string {
	formula := b.formula.Render()

	baseStyle := lipgloss.NewStyle()

	if b.isSelected {
		baseStyle = baseStyle.Inherit(styleBlockSelected)
	}

	if b.isHovered {
		baseStyle = baseStyle.Inherit(styleBlockHovered)
	} else {
		baseStyle = baseStyle.Inherit(styleBlockNormal)
	}

	style := lipgloss.NewStyle().Padding(1).Border(lipgloss.NormalBorder()).Align(lipgloss.Center, lipgloss.Center).Inherit(baseStyle)
	return style.Render(formula)
}

func (b *mathBlock) Toggle() {
	b.isSelected = !b.isSelected
}

type mathTable struct {
	table      [][]mathBlock
	score      int
	hoveredRow int
	hoveredCol int
}

func newMathTable(table [][]mathBlock) mathTable {
	t := mathTable{
		table:      table,
		score:      0,
		hoveredRow: 0,
		hoveredCol: 0,
	}
	t.table[t.hoveredRow][t.hoveredCol].isHovered = true
	return t
}

func (t *mathTable) Render() string {
	rows := make([]string, 0, len(t.table))

	for _, row := range t.table {
		rowString := make([]string, 0, len(row))
		for _, b := range row {
			rowString = append(rowString, b.Render())
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Left, rowString...))
	}

	width := lipgloss.Width(rows[0])
	scoreLabel := lipgloss.NewStyle().Align(lipgloss.Left).Render("score: ")
	scoreValue := lipgloss.NewStyle().Align(lipgloss.Right).Width(width - lipgloss.Width(scoreLabel)).Render(strconv.Itoa(t.score))
	score := lipgloss.JoinHorizontal(lipgloss.Left, scoreLabel, scoreValue)
	rows = append([]string{score}, rows...)

	return lipgloss.JoinVertical(lipgloss.Bottom, rows...)
}

func (t *mathTable) CursorDown() {
	t.table[t.hoveredRow][t.hoveredCol].isHovered = false
	t.hoveredRow = (t.hoveredRow + 1) % len(t.table)
	t.table[t.hoveredRow][t.hoveredCol].isHovered = true
}

func (t *mathTable) CursorUp() {
	t.table[t.hoveredRow][t.hoveredCol].isHovered = false
	t.hoveredRow = (t.hoveredRow + len(t.table) - 1) % len(t.table)
	t.table[t.hoveredRow][t.hoveredCol].isHovered = true
}

func (t *mathTable) CursorRight() {
	t.table[t.hoveredRow][t.hoveredCol].isHovered = false
	t.hoveredCol = (t.hoveredCol + 1) % len(t.table[0])
	t.table[t.hoveredRow][t.hoveredCol].isHovered = true
}

func (t *mathTable) CursorLeft() {
	t.table[t.hoveredRow][t.hoveredCol].isHovered = false
	t.hoveredCol = (t.hoveredCol + len(t.table[0]) - 1) % len(t.table[0])
	t.table[t.hoveredRow][t.hoveredCol].isHovered = true
}

func (t *mathTable) Toggle() {
	t.table[t.hoveredRow][t.hoveredCol].Toggle()
}

type keymap struct {
	up     key.Binding
	down   key.Binding
	left   key.Binding
	right  key.Binding
	choose key.Binding
}

type model struct {
	flexBox *flexbox.FlexBox

	tableLeft  mathTable
	tableRight mathTable

	timer         timer.Model
	timerProgress progress.Model

	keymap keymap
	help   help.Model
}

func newModel(keymap keymap) model {
	mathRows := make([][]mathBlock, 0)
	for i := 0; i < 4; i++ {
		r := make([]mathBlock, 0)
		for j := 0; j < 3; j++ {
			r = append(r, newMathBlock(mathFormula{
				lhs: 1,
				rhs: 1,
				op:  "+",
			}))
		}
		mathRows = append(mathRows, r)
	}

	// TODO: gen another table instead of copying
	mathRows2 := make([][]mathBlock, len(mathRows))
	for i, row := range mathRows {
		mathRows2[i] = make([]mathBlock, len(row))
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
