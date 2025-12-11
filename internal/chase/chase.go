package chase

import (
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"

	"vpet/internal/pet"
)

const (
	tickInterval   = 70 * time.Millisecond
	minVisibleRows = 6

	// Movement speeds in columns per second
	targetSpeedDefault = 8.0  // butterfly moves 8 columns/second
	targetSpeedFast    = 12.0 // mouse moves 12 columns/second
	targetSpeedSlow    = 6.0  // ball moves 6 columns/second
	petSpeed           = 10.0 // pet moves 10 columns/second
)

// RNG is the seeded random number generator for chase mode
// Exposed for testing and future features (pickups, boss targets, etc.)
var RNG *rand.Rand

// getChaseEmoji returns the appropriate emoji for the pet during chase based on its state
func getChaseEmoji(p pet.Pet, distX, distY int) string {
	// Near-catch window: show excitement when the pet closes most of the gap
	absX := distX
	if absX < 0 {
		absX = -absX
	}
	absY := distY
	if absY < 0 {
		absY = -absY
	}
	if absX <= 3 && absY <= 1 {
		return pet.StatusEmojiExcited // Excited about to catch
	}

	// Check hunger level first - critical state takes priority
	if p.Hunger < pet.LowStatThreshold {
		return pet.StatusEmojiHungry // Hungry/desperate
	}

	// Check energy level - affects speed emoji
	if p.Energy < pet.LowStatThreshold {
		return pet.StatusEmojiSleeping // Tired/slow
	} else if p.Energy > pet.AutoWakeEnergy {
		return pet.StatusEmojiEnergetic // Energetic/fast
	}

	// Check happiness level
	if p.Happiness < pet.LowStatThreshold {
		return pet.StatusEmojiSad // Sad/slow
	} else if p.Happiness > pet.HighStatThreshold {
		return pet.StatusEmojiHappy // Very happy
	}

	// Default emoji (neutral)
	return pet.StatusEmojiNeutral
}

// Target defines what the pet can chase
type Target struct {
	Emoji string
	Name  string
	Speed float64 // Columns per second
}

// Available targets (extensible)
var Targets = map[string]Target{
	"butterfly": {Emoji: "🦋", Name: "butterfly", Speed: targetSpeedDefault},
	"ball":      {Emoji: "⚽", Name: "ball", Speed: targetSpeedSlow},
	"mouse":     {Emoji: "🐁", Name: "mouse", Speed: targetSpeedFast},
}

// Model is the Bubble Tea model for chase animation
type Model struct {
	Pet            pet.Pet
	Target         Target
	TermWidth      int
	TermHeight     int
	PetPosX        float64 // Using float64 for smooth delta-time movement
	PetPosY        float64
	TargetPosX     float64
	TargetPosY     float64
	LastUpdateTime time.Time
	ElapsedTime    float64 // Total elapsed time in seconds
}

type animTickMsg time.Time

// Run starts the chase animation
func Run(seed int64) {
	// Initialize RNG with seed (0 = use current time)
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	RNG = rand.New(rand.NewSource(seed))

	p := pet.LoadState()
	target := Targets["butterfly"]

	model := Model{
		Pet:            p,
		Target:         target,
		PetPosX:        0,
		PetPosY:        0,
		TargetPosX:     5,
		TargetPosY:     0,
		LastUpdateTime: time.Now(),
		ElapsedTime:    0,
		TermWidth:      0, // set on first resize event
		TermHeight:     0, // set on first resize event
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
		if m.TermWidth == 0 || m.TermHeight == 0 {
			m.LastUpdateTime = time.Time(msg)
			return m, tick()
		}

		// Calculate delta time since last update
		now := time.Time(msg)
		deltaTime := now.Sub(m.LastUpdateTime).Seconds()
		m.LastUpdateTime = now
		m.ElapsedTime += deltaTime

		// Move target horizontally based on speed
		m.TargetPosX += m.Target.Speed * deltaTime

		if m.TargetPosX >= float64(m.maxX()) {
			return m, tea.Quit
		}

		// Vertical flutter pattern using sine wave
		height := float64(m.visibleRows())
		amplitude := height / 3.0
		centerY := height / 2.0
		frequency := 0.2

		m.TargetPosY = centerY + amplitude*math.Sin(m.TargetPosX*frequency)

		// Move pet - follows butterfly in 2D space
		distX := m.TargetPosX - m.PetPosX
		distY := m.TargetPosY - m.PetPosY

		// Move independently on each axis based on distance thresholds
		if math.Abs(distX) > 3 {
			// Move toward target on X axis
			if distX > 0 {
				m.PetPosX += petSpeed * deltaTime
			} else {
				m.PetPosX -= petSpeed * deltaTime
			}
		}

		if math.Abs(distY) > 1 {
			// Move toward target on Y axis
			if distY > 0 {
				m.PetPosY += petSpeed * deltaTime
			} else {
				m.PetPosY -= petSpeed * deltaTime
			}
		}

		m.clampPositions()

		// Catch condition: overlapping X and same row
		if math.Abs(m.TargetPosX-m.PetPosX) <= 1 && int(m.TargetPosY) == int(m.PetPosY) {
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
	distX := int(m.TargetPosX - m.PetPosX)
	distY := int(m.TargetPosY - m.PetPosY)
	petEmoji := getChaseEmoji(m.Pet, distX, distY)

	// Build 2D grid for animation
	grid := make([][]rune, rows-1)
	for y := 0; y < rows-1; y++ {
		grid[y] = make([]rune, m.TermWidth)
		for x := 0; x < m.TermWidth; x++ {
			grid[y][x] = ' '
		}
	}

	// Helper function to place emoji with proper width handling
	placeEmoji := func(emoji string, x, y int) {
		if y < 0 || y >= rows-1 || x < 0 {
			return
		}

		col := x
		for _, r := range emoji {
			width := runewidth.RuneWidth(r)
			if col+width > m.TermWidth {
				break
			}
			grid[y][col] = r
			col += width
		}
	}

	// Place target at its 2D position (convert float to int for rendering)
	placeEmoji(m.Target.Emoji, int(m.TargetPosX), int(m.TargetPosY))

	// Place pet at its 2D position
	placeEmoji(petEmoji, int(m.PetPosX), int(m.PetPosY))

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

	maxX := float64(m.maxX())
	maxY := float64(rows - 1)

	if m.PetPosX < 0 {
		m.PetPosX = 0
	}
	if m.PetPosX >= maxX {
		m.PetPosX = maxX
	}
	if m.TargetPosX < 0 {
		m.TargetPosX = 0
	}
	if m.TargetPosX >= maxX {
		m.TargetPosX = maxX
	}

	if m.PetPosY < 0 {
		m.PetPosY = 0
	}
	if m.PetPosY >= maxY {
		m.PetPosY = maxY
	}

	if m.TargetPosY < 0 {
		m.TargetPosY = 0
	}
	if m.TargetPosY >= maxY {
		m.TargetPosY = maxY
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
