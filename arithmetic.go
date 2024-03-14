package main

import (
	"strconv"

	"github.com/charmbracelet/lipgloss"
)

type Formula struct {
	lhs int
	rhs int
	op  string
}

func (f *Formula) Render() string {
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

type ArithmeticBlock struct {
	formula    Formula
	value      int
	isSelected bool
	isHovered  bool
}

func NewArithmeticBlock(formula Formula) ArithmeticBlock {
	return ArithmeticBlock{
		formula:    formula,
		value:      0,
		isSelected: false,
		isHovered:  false,
	}
}

func (b *ArithmeticBlock) View() string {
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

func (b *ArithmeticBlock) Toggle() {
	b.isSelected = !b.isSelected
}

type ArithmeticTable struct {
	table      [][]ArithmeticBlock
	score      int
	hoveredRow int
	hoveredCol int
}

func newMathTable(table [][]ArithmeticBlock) *ArithmeticTable {
	t := ArithmeticTable{
		table:      table,
		score:      0,
		hoveredRow: 0,
		hoveredCol: 0,
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

func (t *ArithmeticTable) CursorDown() {
	t.table[t.hoveredRow][t.hoveredCol].isHovered = false
	t.hoveredRow = (t.hoveredRow + 1) % len(t.table)
	t.table[t.hoveredRow][t.hoveredCol].isHovered = true
}

func (t *ArithmeticTable) CursorUp() {
	t.table[t.hoveredRow][t.hoveredCol].isHovered = false
	t.hoveredRow = (t.hoveredRow + len(t.table) - 1) % len(t.table)
	t.table[t.hoveredRow][t.hoveredCol].isHovered = true
}

func (t *ArithmeticTable) CursorRight() {
	t.table[t.hoveredRow][t.hoveredCol].isHovered = false
	t.hoveredCol = (t.hoveredCol + 1) % len(t.table[0])
	t.table[t.hoveredRow][t.hoveredCol].isHovered = true
}

func (t *ArithmeticTable) CursorLeft() {
	t.table[t.hoveredRow][t.hoveredCol].isHovered = false
	t.hoveredCol = (t.hoveredCol + len(t.table[0]) - 1) % len(t.table[0])
	t.table[t.hoveredRow][t.hoveredCol].isHovered = true
}

func (t *ArithmeticTable) Toggle() {
	t.table[t.hoveredRow][t.hoveredCol].Toggle()
}
