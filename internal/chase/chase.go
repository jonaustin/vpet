package chase

import (
	"log"
	"math"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"vpet/internal/pet"
)

// Target defines what the pet can chase
type Target struct {
	Emoji string
	Name  string
	Speed int // Frames to move 1 position
}

// Available targets (extensible)
var Targets = map[string]Target{
	"butterfly": {Emoji: "🦋", Name: "butterfly", Speed: 3},
	"ball":      {Emoji: "⚽", Name: "ball", Speed: 4},
	"mouse":     {Emoji: "🐁", Name: "mouse", Speed: 2},
}

// Model is the Bubble Tea model for chase animation
type Model struct {
	Pet        pet.Pet
	Target     Target
	TermWidth  int
	TermHeight int
	PetPosX    int
	PetPosY    int
	TargetPosX int
	TargetPosY int
	Frame      int
}

type animTickMsg time.Time

// Run starts the chase animation
func Run() {
	p := pet.LoadState()
	target := Targets["butterfly"]

	model := Model{
		Pet:        p,
		Target:     target,
		PetPosX:    0,
		PetPosY:    12,
		TargetPosX: 5,
		TargetPosY: 12,
		Frame:      0,
		TermWidth:  80,
		TermHeight: 24,
	}

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		log.Printf("Chase animation error: %v", err)
		os.Exit(1)
	}
}

func tick() tea.Cmd {
	return tea.Tick(70*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tick(),
		tea.EnterAltScreen,
	)
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.TermWidth = msg.Width
		m.TermHeight = msg.Height
		return m, nil

	case animTickMsg:
		m.Frame++

		// Move target (butterfly) - moves every N frames based on speed
		if m.Frame%m.Target.Speed == 0 {
			m.TargetPosX++

			if m.TargetPosX >= m.TermWidth-2 {
				return m, tea.Quit
			}

			// Vertical flutter pattern using sine wave
			amplitude := float64(m.TermHeight) / 4.0
			centerY := float64(m.TermHeight) / 2.0
			frequency := 0.2

			newY := centerY + amplitude*math.Sin(float64(m.TargetPosX)*frequency)
			m.TargetPosY = int(newY)

			if m.TargetPosY < 2 {
				m.TargetPosY = 2
			}
			if m.TargetPosY > m.TermHeight-5 {
				m.TargetPosY = m.TermHeight - 5
			}
		}

		// Move pet - follows butterfly in 2D space
		if m.Frame%2 == 0 {
			distX := m.TargetPosX - m.PetPosX
			distY := m.TargetPosY - m.PetPosY

			if distX > 3 {
				m.PetPosX++
			}

			if distY > 1 {
				m.PetPosY++
			} else if distY < -1 {
				m.PetPosY--
			}

			if m.PetPosY < 2 {
				m.PetPosY = 2
			}
			if m.PetPosY > m.TermHeight-5 {
				m.PetPosY = m.TermHeight - 5
			}
		}

		return m, tick()
	}

	return m, nil
}

// View implements tea.Model
func (m Model) View() string {
	if m.TermWidth == 0 || m.TermHeight == 0 {
		return "Initializing..."
	}

	petEmoji := "😸"

	// Build 2D grid for animation
	grid := make([][]rune, m.TermHeight)
	for y := 0; y < m.TermHeight; y++ {
		grid[y] = make([]rune, m.TermWidth)
		for x := 0; x < m.TermWidth; x++ {
			grid[y][x] = ' '
		}
	}

	// Place target at its 2D position
	if m.TargetPosY >= 0 && m.TargetPosY < m.TermHeight && m.TargetPosX >= 0 && m.TargetPosX < m.TermWidth-2 {
		targetRunes := []rune(m.Target.Emoji)
		for i, r := range targetRunes {
			if m.TargetPosX+i < m.TermWidth {
				grid[m.TargetPosY][m.TargetPosX+i] = r
			}
		}
	}

	// Place pet at its 2D position
	if m.PetPosY >= 0 && m.PetPosY < m.TermHeight && m.PetPosX >= 0 && m.PetPosX < m.TermWidth-2 {
		petRunes := []rune(petEmoji)
		for i, r := range petRunes {
			if m.PetPosX+i < m.TermWidth {
				grid[m.PetPosY][m.PetPosX+i] = r
			}
		}
	}

	// Convert grid to string
	var result strings.Builder
	for y := 0; y < m.TermHeight-3; y++ {
		result.WriteString(string(grid[y]))
		result.WriteRune('\n')
	}

	result.WriteString("\nPress any key to exit")

	return result.String()
}
