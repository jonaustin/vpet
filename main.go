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
	pet     Pet
	choice  int
	quitting bool
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
		// Hunger decreases every 15 minutes
		hungerLoss := ((hours * 60) + minutes) / 15 * 5
		pet.Hunger = max(pet.Hunger-hungerLoss, 0)

		// Energy decreases every 30 minutes
		energyLoss := ((hours * 60) + minutes) / 30 * 5
		pet.Energy = max(pet.Energy-energyLoss, 0)

		// Happiness affected by low stats
		if pet.Hunger < 30 || pet.Energy < 30 {
			happinessLoss := ((hours * 60) + minutes) / 15 * 2
			pet.Happiness = max(pet.Happiness-happinessLoss, 0)
		}
	} else {
		// Sleeping recovers energy
		energyGain := ((hours * 60) + minutes) / 15 * 10
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
				m.pet.Hunger = min(m.pet.Hunger+30, 100)
				m.pet.Happiness = min(m.pet.Happiness+10, 100)
			case 1: // Play
				if !m.pet.Sleeping {
					m.pet.Happiness = min(m.pet.Happiness+30, 100)
					m.pet.Energy = max(m.pet.Energy-20, 0)
					m.pet.Hunger = max(m.pet.Hunger-10, 0)
				}
			case 2: // Sleep
				m.pet.Sleeping = !m.pet.Sleeping
			case 3: // Quit
				m.quitting = true
				return m, tea.Quit
			}
		}

	case tickMsg:
		if !m.pet.Sleeping {
			// Hunger decreases every 15 minutes
			if int(time.Time(msg).Minute())%15 == 0 {
				m.pet.Hunger = max(m.pet.Hunger-5, 0)
			}
			// Energy decreases every 30 minutes
			if int(time.Time(msg).Minute())%30 == 0 {
				m.pet.Energy = max(m.pet.Energy-5, 0)
			}
			// Happiness affected by hunger and energy
			if m.pet.Hunger < 30 || m.pet.Energy < 30 {
				m.pet.Happiness = max(m.pet.Happiness-2, 0)
			}
		} else {
			// Sleeping recovers energy faster
			if int(time.Time(msg).Minute())%15 == 0 {
				m.pet.Energy = min(m.pet.Energy+10, 100)
				if m.pet.Energy >= 100 {
					m.pet.Sleeping = false
				}
			}
		}
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
	choices := []string{"Feed", "Play", "Sleep", "Quit"}
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
	m, err := p.Run()
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	
	// Save state when quitting
	if m, ok := m.(model); ok && m.quitting {
		saveState(m.pet)
	}
}
