package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Pet struct {
	name      string
	hunger    int
	happiness int
	energy    int
	sleeping  bool
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

func initialModel() model {
	return model{
		pet: Pet{
			name:      "Charm Pet",
			hunger:    100,
			happiness: 100,
			energy:    100,
			sleeping:  false,
		},
		choice: 0,
	}
}

func (m model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
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
				m.pet.hunger = min(m.pet.hunger+30, 100)
				m.pet.happiness = min(m.pet.happiness+10, 100)
			case 1: // Play
				if !m.pet.sleeping {
					m.pet.happiness = min(m.pet.happiness+30, 100)
					m.pet.energy = max(m.pet.energy-20, 0)
					m.pet.hunger = max(m.pet.hunger-10, 0)
				}
			case 2: // Sleep
				m.pet.sleeping = !m.pet.sleeping
			case 3: // Quit
				m.quitting = true
				return m, tea.Quit
			}
		}

	case tickMsg:
		if !m.pet.sleeping {
			m.pet.hunger = max(m.pet.hunger-1, 0)
			m.pet.energy = max(m.pet.energy-1, 0)
			if m.pet.hunger < 30 || m.pet.energy < 30 {
				m.pet.happiness = max(m.pet.happiness-1, 0)
			}
		} else {
			m.pet.energy = min(m.pet.energy+2, 100)
			if m.pet.energy >= 100 {
				m.pet.sleeping = false
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

	s := titleStyle.Render("🐱 " + m.pet.name + " 🐱\n\n")

	// Status
	s += statusStyle.Render(fmt.Sprintf("Hunger:    %d%%\n", m.pet.hunger))
	s += statusStyle.Render(fmt.Sprintf("Happiness: %d%%\n", m.pet.happiness))
	s += statusStyle.Render(fmt.Sprintf("Energy:    %d%%\n", m.pet.energy))
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
	if p.sleeping {
		return "😴 Sleeping"
	}
	if p.hunger < 30 {
		return "😫 Hungry"
	}
	if p.energy < 30 {
		return "😩 Tired"
	}
	if p.happiness < 30 {
		return "😢 Sad"
	}
	return "😊 Happy"
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
