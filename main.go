package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Pet struct {
	Name      string    `json:"name"`
	Hunger    int       `json:"hunger"`
	Happiness int       `json:"happiness"`
	Energy    int       `json:"energy"`
	Sleeping  bool      `json:"sleeping"`
	LastSaved time.Time `json:"last_saved"`
}

type model struct {
	pet      Pet
	choice   int
	quitting bool
}

// Helper function to modify stats and save immediately
func (m *model) modifyStats(f func(*Pet)) {
	f(&m.pet)
	saveState(m.pet)
}

// Pet state modification functions
func (m *model) feed() {
	m.modifyStats(func(p *Pet) {
		p.Hunger = min(p.Hunger+30, 100)
		p.Happiness = min(p.Happiness+10, 100)
	})
}

func (m *model) play() {
	if !m.pet.Sleeping {
		m.modifyStats(func(p *Pet) {
			p.Happiness = min(p.Happiness+30, 100)
			p.Energy = max(p.Energy-20, 0)
			p.Hunger = max(p.Hunger-10, 0)
		})
	}
}

func (m *model) toggleSleep() {
	m.modifyStats(func(p *Pet) {
		p.Sleeping = !p.Sleeping
	})
}

func (m *model) updateHourlyStats(t time.Time) {
	if !m.pet.Sleeping {
		m.modifyStats(func(p *Pet) {
			// Hunger decreases every hour
			if int(t.Minute())%60 == 0 {
				p.Hunger = max(p.Hunger-5, 0)
			}
			// Energy decreases every 2 hours
			if int(t.Minute())%120 == 0 {
				p.Energy = max(p.Energy-5, 0)
			}
			// Happiness affected by hunger and energy
			if p.Hunger < 30 || p.Energy < 30 {
				if int(t.Minute())%60 == 0 {
					p.Happiness = max(p.Happiness-2, 0)
				}
			}
		})
	} else {
		// Sleeping recovers energy faster
		if int(t.Minute())%60 == 0 {
			m.modifyStats(func(p *Pet) {
				p.Energy = min(p.Energy+10, 100)
				if p.Energy >= 100 {
					p.Sleeping = false
				}
			})
		}
	}
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF75B5")).
			MarginBottom(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF75B5"))

	menuStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF75B5"))
)

func loadState() Pet {
	data, err := os.ReadFile("pet.json")
	if err != nil {
		return Pet{
			Name:      "Charm Pet",
			Hunger:    100,
			Happiness: 100,
			Energy:    100,
			Sleeping:  false,
			LastSaved: time.Now(),
		}
	}

	var pet Pet
	if err := json.Unmarshal(data, &pet); err != nil {
		fmt.Printf("Error loading state: %v\n", err)
		os.Exit(1)
	}

	// Calculate state changes based on elapsed time
	elapsed := time.Since(pet.LastSaved)
	hours := int(elapsed.Hours())
	minutes := int(elapsed.Minutes()) % 60

	if !pet.Sleeping {
		// Hunger decreases every hour
		hungerLoss := ((hours * 60) + minutes) / 60 * 5
		pet.Hunger = max(pet.Hunger-hungerLoss, 0)

		// Energy decreases every 2 hours
		energyLoss := ((hours * 60) + minutes) / 120 * 5
		pet.Energy = max(pet.Energy-energyLoss, 0)

		// Happiness affected by low stats
		if pet.Hunger < 30 || pet.Energy < 30 {
			happinessLoss := ((hours * 60) + minutes) / 60 * 2
			pet.Happiness = max(pet.Happiness-happinessLoss, 0)
		}
	} else {
		// Sleeping recovers energy
		energyGain := ((hours * 60) + minutes) / 60 * 10
		pet.Energy = min(pet.Energy+energyGain, 100)
		if pet.Energy >= 100 {
			pet.Sleeping = false
		}
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
	if err := os.WriteFile("pet.json", data, 0644); err != nil {
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

	s := titleStyle.Render("😺 " + m.pet.Name + " 😺\n\n")

	// Status
	s += statusStyle.Render(fmt.Sprintf("Hunger:    %d%%\n", m.pet.Hunger))
	s += statusStyle.Render(fmt.Sprintf("Happiness: %d%%\n", m.pet.Happiness))
	s += statusStyle.Render(fmt.Sprintf("Energy:    %d%%\n", m.pet.Energy))
	s += statusStyle.Render(fmt.Sprintf("Status:    %s\n\n", getStatus(m.pet)))

	// Menu
	choices := []string{"Play", "Feed", "Sleep", "Quit"}
	for i, choice := range choices {
		cursor := " "
		if m.choice == i {
			cursor = ">"
		}
		s += menuStyle.Render(fmt.Sprintf("%s %s\n", cursor, choice))
	}

	s += "\n" + statusStyle.Render("Use ↑/↓ to select, enter to confirm")
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
