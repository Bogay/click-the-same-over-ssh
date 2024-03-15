package main

import (
	"math/rand"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

const (
	OperatorPlug = "+"
)

type Formula struct {
	lhs int
	rhs int
	op  string

	val int
}

func NewFormula(val int) *Formula {
	lhs := rand.Intn(val)
	rhs := val - lhs

	return &Formula{
		lhs: lhs,
		rhs: rhs,
		op:  OperatorPlug,
		val: val,
	}
}

func (f *Formula) View() string {
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

func (f *Formula) UpdateValue(val int) {
	lhs := rand.Intn(val)
	rhs := val - lhs

	f.lhs = lhs
	f.rhs = rhs
	f.val = val
}

func (f *Formula) Value() int {
	return f.val
}

type ArithmeticBlock struct {
	formula    *Formula
	isSelected bool
	isHovered  bool
}

func NewArithmeticBlock(val int) ArithmeticBlock {
	return ArithmeticBlock{
		formula:    NewFormula(val),
		isSelected: false,
		isHovered:  false,
	}
}

func (b *ArithmeticBlock) View() string {
	formula := b.formula.View()

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

func (b *ArithmeticBlock) Toggle() *ArithmeticBlock {
	b.isSelected = !b.isSelected
	return b
}

func (b *ArithmeticBlock) Value() int {
	return b.formula.Value()
}

func (b *ArithmeticBlock) UpdateValue(val int) {
	b.formula.UpdateValue(val)
	b.isSelected = false
}

type ArithmeticTable struct {
	table      [][]ArithmeticBlock
	score      int
	hoveredRow int
	hoveredCol int

	selectedBlock *ArithmeticBlock
	selectedRow   int
	selectedCol   int

	updateBlockFlagsCh chan BlockFlags
}

func newMathTable(table [][]ArithmeticBlock) *ArithmeticTable {
	t := ArithmeticTable{
		table:              table,
		score:              0,
		hoveredRow:         0,
		hoveredCol:         0,
		updateBlockFlagsCh: make(chan BlockFlags),
	}
	t.table[t.hoveredRow][t.hoveredCol].isHovered = true
	return &t
}

func (t *ArithmeticTable) Render() string {
	rows := make([]string, 0, len(t.table))

	for _, row := range t.table {
		rowString := make([]string, 0, len(row))
		for _, b := range row {
			rowString = append(rowString, b.View())
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

func (t *ArithmeticTable) updateCursor(updater func()) {
	t.table[t.hoveredRow][t.hoveredCol].isHovered = false
	t.updateBlockFlagsCh <- updateBlockFlags(t.hoveredRow, t.hoveredCol, &t.table[t.hoveredRow][t.hoveredCol])

	updater()

	t.table[t.hoveredRow][t.hoveredCol].isHovered = true
	t.updateBlockFlagsCh <- updateBlockFlags(t.hoveredRow, t.hoveredCol, &t.table[t.hoveredRow][t.hoveredCol])
}

func (t *ArithmeticTable) CursorDown() {
	t.updateCursor(func() {
		t.hoveredRow = (t.hoveredRow + 1) % len(t.table)
	})
}

func (t *ArithmeticTable) CursorUp() {
	t.updateCursor(func() {
		t.hoveredRow = (t.hoveredRow + len(t.table) - 1) % len(t.table)
	})
}

func (t *ArithmeticTable) CursorRight() {
	t.updateCursor(func() {
		t.hoveredCol = (t.hoveredCol + 1) % len(t.table[0])
	})
}

func (t *ArithmeticTable) CursorLeft() {
	t.updateCursor(func() {
		t.hoveredCol = (t.hoveredCol + len(t.table[0]) - 1) % len(t.table[0])
	})
}

func (t *ArithmeticTable) Toggle() int {
	b := t.table[t.hoveredRow][t.hoveredCol].Toggle()
	t.updateBlockFlagsCh <- updateBlockFlags(t.hoveredRow, t.hoveredCol, b)

	score := 0
	if t.selectedBlock == nil {
		t.selectedBlock = b
		t.selectedRow = t.hoveredRow
		t.selectedCol = t.hoveredCol
	} else {
		a := t.selectedBlock

		if a.Value() == b.Value() {
			a.UpdateValue(1 + rand.Intn(13))
			b.UpdateValue(1 + rand.Intn(13))
			score = 1
		} else {
			a.Toggle()
			b.Toggle()

			log.Debugf("wrong")
		}
		t.updateBlockFlagsCh <- updateBlockFlags(t.selectedRow, t.selectedCol, a)
		t.updateBlockFlagsCh <- updateBlockFlags(t.hoveredRow, t.hoveredCol, b)
		t.selectedBlock = nil
	}

	return score
}
