package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"vpet/internal/pet"
)

// StatsModel is a simple Bubble Tea model for displaying stats
type StatsModel struct {
	Pet pet.Pet
}

// Init implements tea.Model
func (m StatsModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m StatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model
func (m StatsModel) View() string {
	makeBar := func(value int) string {
		filled := value / 20
		bar := ""
		for i := 0; i < 5; i++ {
			if i < filled {
				bar += "█"
			} else {
				bar += "░"
			}
		}
		return bar
	}

	formEmoji := m.Pet.GetFormEmoji()
	formName := m.Pet.GetFormName()
	status := pet.GetStatus(m.Pet)
	illnessStatus := "No"
	if m.Pet.Illness {
		illnessStatus = "Yes"
	}

	chronoEmoji := pet.GetChronotypeEmoji(m.Pet.Chronotype)
	chronoName := pet.GetChronotypeName(m.Pet.Chronotype)
	wakeHour, sleepHour := pet.GetChronotypeSchedule(m.Pet.Chronotype)
	chronoDisplay := fmt.Sprintf("%s %s (%d:00-%d:00)", chronoEmoji, chronoName, wakeHour, sleepHour)

	var traitNames []string
	for _, trait := range m.Pet.Traits {
		traitNames = append(traitNames, trait.Name)
	}
	traitDisplay := strings.Join(traitNames, ", ")
	if traitDisplay == "" {
		traitDisplay = "None"
	}

	bondDisplay := pet.GetBondDescription(m.Pet.Bond)

	var s strings.Builder
	s.WriteString("╔════════════════════════════════════╗\n")
	s.WriteString(fmt.Sprintf("║  %s %s %s                  ║\n", formEmoji, m.Pet.Name, formEmoji))
	s.WriteString("╠════════════════════════════════════╣\n")
	s.WriteString(fmt.Sprintf("║  Form:    %-24s ║\n", formName))
	s.WriteString(fmt.Sprintf("║  Type:    %-24s ║\n", chronoDisplay))
	s.WriteString(fmt.Sprintf("║  Traits:  %-24s ║\n", traitDisplay))
	s.WriteString(fmt.Sprintf("║  Bond:    %-24s ║\n", bondDisplay))
	s.WriteString(fmt.Sprintf("║  Age:     %-24s ║\n", fmt.Sprintf("%d hours", m.Pet.Age)))
	s.WriteString(fmt.Sprintf("║  Status:  %-24s ║\n", status))
	s.WriteString("║                                    ║\n")
	s.WriteString(fmt.Sprintf("║  Hunger:    [%s] %3d%%           ║\n", makeBar(m.Pet.Hunger), m.Pet.Hunger))
	s.WriteString(fmt.Sprintf("║  Happiness: [%s] %3d%%           ║\n", makeBar(m.Pet.Happiness), m.Pet.Happiness))
	s.WriteString(fmt.Sprintf("║  Energy:    [%s] %3d%%           ║\n", makeBar(m.Pet.Energy), m.Pet.Energy))
	s.WriteString(fmt.Sprintf("║  Health:    [%s] %3d%%           ║\n", makeBar(m.Pet.Health), m.Pet.Health))
	s.WriteString("║                                    ║\n")
	s.WriteString(fmt.Sprintf("║  Illness:   %-23s║\n", illnessStatus))
	s.WriteString("╚════════════════════════════════════╝\n")
	s.WriteString("\nPress ESC, click, or any key to close...")

	return s.String()
}

// DisplayStats shows the stats display
func DisplayStats(p pet.Pet) {
	program := tea.NewProgram(StatsModel{Pet: p}, tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := program.Run(); err != nil {
		fmt.Printf("Error running stats display: %v\n", err)
		os.Exit(1)
	}
}
