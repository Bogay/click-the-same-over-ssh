package main

import (
	"time"

	"github.com/charmbracelet/lipgloss"
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
