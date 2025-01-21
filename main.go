package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Game constants
const (
	defaultPetName = "Charm Pet"
	maxStat       = 100
	minStat       = 0
	lowStatThreshold = 30
	
	// Stat change rates
	hungerDecreaseRate    = 5
	sleepingHungerRate    = 3  // 70% of normal rate
	energyDecreaseRate    = 5
	energyRecoveryRate    = 10
	happinessDecreaseRate = 2
	
	feedHungerIncrease    = 30
	feedHappinessIncrease = 10
	playHappinessIncrease = 30
	playEnergyDecrease    = 20
	playHungerDecrease    = 10
)

// Pet represents the virtual pet's state
type Pet struct {
	Name      string    `json:"name"`
	Hunger    int       `json:"hunger"`
	Happiness int       `json:"happiness"`
	Energy    int       `json:"energy"`
	Sleeping  bool      `json:"sleeping"`
	LastSaved time.Time `json:"last_saved"`
}

// model represents the game state
type model struct {
	pet      Pet
	choice   int
	quitting bool
}

// UI styles
type styles struct {
	title  lipgloss.Style
	status lipgloss.Style
	menu   lipgloss.Style
}

// Helper function to modify stats and save immediately
func (m *model) modifyStats(f func(*Pet)) {
	f(&m.pet)
	saveState(m.pet)
}

// Pet state modification functions
// feed increases hunger and happiness
func (m *model) feed() {
	m.modifyStats(func(p *Pet) {
		p.Sleeping = false
		p.Hunger = min(p.Hunger+feedHungerIncrease, maxStat)
		p.Happiness = min(p.Happiness+feedHappinessIncrease, maxStat)
	})
}

// play increases happiness but decreases energy and hunger
func (m *model) play() {
	m.modifyStats(func(p *Pet) {
		p.Sleeping = false
		p.Happiness = min(p.Happiness+playHappinessIncrease, maxStat)
		p.Energy = max(p.Energy-playEnergyDecrease, minStat)
		p.Hunger = max(p.Hunger-playHungerDecrease, minStat)
	})
}

func (m *model) toggleSleep() {
	m.modifyStats(func(p *Pet) {
		p.Sleeping = !p.Sleeping
	})
}

func (m *model) updateHourlyStats(t time.Time) {
	m.modifyStats(func(p *Pet) {
		// Hunger decreases every hour (reduced rate while sleeping)
		if int(t.Minute())%60 == 0 {
			hungerRate := 5
			if p.Sleeping {
				hungerRate = 3 // 70% of 5 rounded down
			}
			p.Hunger = max(p.Hunger-hungerRate, 0)
		}

		if !p.Sleeping {
			// Energy decreases every 2 hours when awake
			if int(t.Minute())%120 == 0 {
				p.Energy = max(p.Energy-5, 0)
			}
		} else {
			// Sleeping recovers energy faster
			if int(t.Minute())%60 == 0 {
				p.Energy = min(p.Energy+10, 100)
				if p.Energy >= 100 {
					p.Sleeping = false
				}
			}
		}

		// Happiness affected by hunger and energy
		if p.Hunger < 30 || p.Energy < 30 {
			if int(t.Minute())%60 == 0 {
				p.Happiness = max(p.Happiness-2, 0)
			}
		}
	})
}

var (
	gameStyles = styles{
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF75B5")).
			MarginBottom(1),
		
		status: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF75B5")),
		
		menu: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF75B5")),
	}
)

// newPet creates a new pet with default values
func newPet() Pet {
	return Pet{
		Name:      defaultPetName,
		Hunger:    maxStat,
		Happiness: maxStat,
		Energy:    maxStat,
		Sleeping:  false,
		LastSaved: time.Now(),
	}
}

// loadState loads the pet's state from file or creates a new pet
func getConfigPath() string {
	configDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(configDir, ".config", "vpet", "pet.json")
}

func loadState() Pet {
	configPath := getConfigPath()
	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return newPet()
	}

	var pet Pet
	if err := json.Unmarshal(data, &pet); err != nil {
		fmt.Printf("Error loading state: %v\n", err)
		os.Exit(1)
	}

	// Update stats based on elapsed time
	elapsed := time.Since(pet.LastSaved)
	totalMinutes := int(elapsed.Hours())*60 + int(elapsed.Minutes())%60

	// Calculate hunger decrease
	hungerRate := hungerDecreaseRate
	if pet.Sleeping {
		hungerRate = sleepingHungerRate
	}
	hungerLoss := (totalMinutes / 60) * hungerRate
	pet.Hunger = max(pet.Hunger-hungerLoss, minStat)

	if !pet.Sleeping {
		// Energy decreases when awake
		energyLoss := (totalMinutes / 120) * energyDecreaseRate
		pet.Energy = max(pet.Energy-energyLoss, minStat)
	} else {
		// Energy recovers while sleeping
		energyGain := (totalMinutes / 60) * energyRecoveryRate
		pet.Energy = min(pet.Energy+energyGain, maxStat)
		if pet.Energy >= maxStat {
			pet.Sleeping = false
		}
	}

	// Update happiness if stats are low
	if pet.Hunger < lowStatThreshold || pet.Energy < lowStatThreshold {
		happinessLoss := (totalMinutes / 60) * happinessDecreaseRate
		pet.Happiness = max(pet.Happiness-happinessLoss, minStat)
	}

	pet.LastSaved = time.Now()
	return pet
}

func saveState(p Pet) {
	p.LastSaved = time.Now()
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		fmt.Printf("Error saving state: %v\n", err)
		return
	}
	if err := os.WriteFile(getConfigPath(), data, 0644); err != nil {
		fmt.Printf("Error writing state: %v\n", err)
	}
}

func initialModel() model {
	return model{
		pet:    loadState(),
		choice: 0,
	}
}

func (m model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(time.Minute, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.choice > 0 {
				m.choice--
			}
		case "down", "j":
			if m.choice < 3 {
				m.choice++
			}
		case "enter", " ":
			switch m.choice {
			case 0: // Feed
				m.feed()
			case 1: // Play
				m.play()
			case 2: // Sleep
				m.toggleSleep()
			case 3: // Quit
				m.quitting = true
				return m, tea.Quit
			}
		}

	case tickMsg:
		m.updateHourlyStats(time.Time(msg))
		return m, tick()
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Thanks for playing!\n"
	}

	s := gameStyles.title.Render("😺 " + m.pet.Name + " 😺\n\n")

	// Status display
	stats := []struct {
		name  string
		value int
	}{
		{"Hunger", m.pet.Hunger},
		{"Happiness", m.pet.Happiness},
		{"Energy", m.pet.Energy},
	}

	for _, stat := range stats {
		s += gameStyles.status.Render(fmt.Sprintf("%-10s %d%%\n", stat.name+":", stat.value))
	}
	s += gameStyles.status.Render(fmt.Sprintf("%-10s %s\n\n", "Status:", getStatus(m.pet)))

	// Menu display
	choices := []string{"Feed", "Play", "Sleep", "Quit"}
	for i, choice := range choices {
		cursor := " "
		if m.choice == i {
			cursor = ">"
		}
		s += gameStyles.menu.Render(fmt.Sprintf("      %s %s\n", cursor, choice))
	}

	s += "\n" + gameStyles.status.Render("Use ↑/↓ to select, enter to confirm")
	return s
}

func getStatus(p Pet) string {
	if p.Sleeping {
		return "😴 Sleeping"
	}
	if p.Hunger < 30 {
		return "🙀 Hungry"
	}
	if p.Energy < 30 {
		return "😾 Tired"
	}
	if p.Happiness < 30 {
		return "😿 Sad"
	}
	return "😸 Happy"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

}
