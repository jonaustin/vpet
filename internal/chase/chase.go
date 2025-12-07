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

const (
	tickInterval   = 70 * time.Millisecond
	minVisibleRows = 6
)

// getChaseEmoji returns the appropriate emoji for the pet during chase based on its state
func getChaseEmoji(p pet.Pet, distX, distY int) string {
	// Check if pet is about to catch (very close)
	if absInt(distX) <= 2 && absInt(distY) <= 1 {
		return "😻" // Excited about to catch
	}

	// Check energy level - affects speed emoji
	if p.Energy < 30 {
		return "😴" // Tired/slow
	} else if p.Energy > 80 {
		return "😼" // Energetic/fast
	}

	// Check hunger level
	if p.Hunger < 30 {
		return "🙀" // Hungry/desperate
	}

	// Check happiness level
	if p.Happiness < 30 {
		return "😿" // Sad/slow
	} else if p.Happiness > 80 {
		return "😸" // Default happy
	}

	// Default emoji
	return "😸"
}

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
		PetPosY:    0,
		TargetPosX: 5,
		TargetPosY: 0,
		Frame:      0,
		TermWidth:  0, // set on first resize event
		TermHeight: 0, // set on first resize event
	}

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		log.Printf("Chase animation error: %v", err)
		os.Exit(1)
	}
}

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
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
		m.clampPositions()
		return m, nil

	case animTickMsg:
		m.Frame++

		if m.TermWidth == 0 || m.TermHeight == 0 {
			return m, tick()
		}

		// Move target (butterfly) - moves every N frames based on speed
		if m.Frame%m.Target.Speed == 0 {
			m.TargetPosX++

			if m.TargetPosX >= m.maxX() {
				return m, tea.Quit
			}

			// Vertical flutter pattern using sine wave
			height := float64(m.visibleRows())
			amplitude := height / 3.0
			centerY := height / 2.0
			frequency := 0.2

			newY := centerY + amplitude*math.Sin(float64(m.TargetPosX)*frequency)
			m.TargetPosY = int(newY)

			m.clampPositions()
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

			m.clampPositions()
		}

		// Catch condition: overlapping X and same row
		if absInt(m.TargetPosX-m.PetPosX) <= 1 && m.TargetPosY == m.PetPosY {
			return m, tea.Quit
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

	rows := m.visibleRows()

	// Calculate distance to determine emoji
	distX := m.TargetPosX - m.PetPosX
	distY := m.TargetPosY - m.PetPosY
	petEmoji := getChaseEmoji(m.Pet, distX, distY)

	// Build 2D grid for animation
	grid := make([][]rune, rows-1)
	for y := 0; y < rows-1; y++ {
		grid[y] = make([]rune, m.TermWidth)
		for x := 0; x < m.TermWidth; x++ {
			grid[y][x] = ' '
		}
	}

	// Place target at its 2D position
	if m.TargetPosY >= 0 && m.TargetPosY < rows && m.TargetPosX >= 0 && m.TargetPosX < m.TermWidth-2 {
		targetRunes := []rune(m.Target.Emoji)
		for i, r := range targetRunes {
			if m.TargetPosX+i < m.TermWidth {
				grid[m.TargetPosY][m.TargetPosX+i] = r
			}
		}
	}

	// Place pet at its 2D position
	if m.PetPosY >= 0 && m.PetPosY < rows && m.PetPosX >= 0 && m.PetPosX < m.TermWidth-2 {
		petRunes := []rune(petEmoji)
		for i, r := range petRunes {
			if m.PetPosX+i < m.TermWidth {
				grid[m.PetPosY][m.PetPosX+i] = r
			}
		}
	}

	// Convert grid to string
	var result strings.Builder
	for y := 0; y < rows-1; y++ {
		result.WriteString(string(grid[y]))
		result.WriteRune('\n')
	}

	result.WriteString("\nPress any key to exit")

	return result.String()
}

func (m *Model) clampPositions() {
	rows := m.visibleRows()
	if rows < 1 {
		return
	}

	if m.PetPosX < 0 {
		m.PetPosX = 0
	}
	if m.PetPosX >= m.maxX() {
		m.PetPosX = m.maxX()
	}
	if m.TargetPosX < 0 {
		m.TargetPosX = 0
	}
	if m.TargetPosX >= m.maxX() {
		m.TargetPosX = m.maxX()
	}

	if m.PetPosY < 0 {
		m.PetPosY = 0
	}
	if m.PetPosY >= rows {
		m.PetPosY = rows - 1
	}

	if m.TargetPosY < 0 {
		m.TargetPosY = 0
	}
	if m.TargetPosY >= rows {
		m.TargetPosY = rows - 1
	}
}

func (m Model) visibleRows() int {
	if m.TermHeight <= 0 {
		return 0
	}
	rows := m.TermHeight - 2 // leave space for instruction
	if rows < minVisibleRows {
		rows = minVisibleRows
	}
	return rows
}

func (m Model) maxX() int {
	if m.TermWidth <= 2 {
		return 0
	}
	return m.TermWidth - 2
}
