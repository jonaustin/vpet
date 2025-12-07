package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"vpet/internal/pet"
)

var gameStyles = struct {
	title   lipgloss.Style
	status  lipgloss.Style
	menu    lipgloss.Style
	menuBox lipgloss.Style
	stats   lipgloss.Style
}{
	title: lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF75B5")).
		Padding(0, 1),

	status: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF75B5")).
		Width(30),

	stats: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF75B5")).
		Width(30),

	menu: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF75B5")),

	menuBox: lipgloss.NewStyle().
		Padding(0, 2),
}

// View implements tea.Model
func (m Model) View() string {
	if m.Pet.Dead {
		return m.deadView()
	}
	if m.Quitting {
		return "Thanks for playing!\n"
	}
	if m.InCheatMenu {
		return m.renderCheatMenu()
	}

	// Show animation if one is active
	if m.Animation.Type != AnimNone {
		return m.renderAnimation()
	}

	formEmoji := m.Pet.GetFormEmoji()
	title := gameStyles.title.Render(formEmoji + " " + m.Pet.Name + " " + formEmoji)
	stats := m.renderStats()
	status := m.renderStatus()
	menu := m.renderMenu()

	var messageView string
	if m.Message != "" && pet.TimeNow().Before(m.MessageExpires) {
		messageView = gameStyles.status.Render(m.Message)
	}

	var eventView string
	emoji, eventMsg, hasEvent := m.Pet.GetEventDisplay()
	if hasEvent {
		eventView = gameStyles.title.Render(fmt.Sprintf("✨ %s %s %s ✨", emoji, m.Pet.Name+" is "+eventMsg, emoji))
	}

	sections := []string{
		title,
		"",
		stats,
		"",
		status,
	}

	if eventView != "" {
		sections = append(sections, "", eventView, gameStyles.status.Render("Press [E] to respond!"))
	}

	if messageView != "" {
		sections = append(sections, "", messageView)
	}

	helpText := "Use arrows to move • enter to select • q to quit"
	if hasEvent {
		helpText = "[E] Respond to event • arrows to move • enter to select • q to quit"
	}

	sections = append(sections,
		"",
		menu,
		"",
		gameStyles.status.Render(helpText),
	)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderStats() string {
	mood := m.Pet.Mood
	if mood == "" {
		mood = "normal"
	}
	moodDisplay := strings.ToUpper(mood[:1]) + mood[1:]

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

	stats := []struct {
		name, value string
	}{
		{"Form", m.Pet.GetFormName()},
		{"Type", chronoDisplay},
		{"Traits", traitDisplay},
		{"Bond", pet.GetBondDescription(m.Pet.Bond)},
		{"Mood", moodDisplay},
		{"Hunger", fmt.Sprintf("%d%%", m.Pet.Hunger)},
		{"Happiness", fmt.Sprintf("%d%%", m.Pet.Happiness)},
		{"Energy", fmt.Sprintf("%d%%", m.Pet.Energy)},
		{"Health", fmt.Sprintf("%d%%", m.Pet.Health)},
		{"Age", fmt.Sprintf("%dh", m.Pet.Age)},
		{"Illness", map[bool]string{true: "Yes", false: "No"}[m.Pet.Illness]},
	}

	var lines []string
	for _, stat := range stats {
		lines = append(lines, fmt.Sprintf("%-10s %s", stat.name+":", stat.value))
	}

	return gameStyles.stats.Render(strings.Join(lines, "\n"))
}

func (m Model) renderStatus() string {
	return gameStyles.status.Render(fmt.Sprintf("Status: %s", pet.GetStatusWithLabel(m.Pet)))
}

func (m Model) renderMenu() string {
	choices := []string{"Feed", "Play", "Sleep", "Medicine", "Quit"}
	var menuItems []string

	for i, choice := range choices {
		cursor := " "
		if m.Choice == i {
			cursor = ">"
		}
		menuItems = append(menuItems, fmt.Sprintf("%s %s", cursor, choice))
	}

	return gameStyles.menuBox.Render(strings.Join(menuItems, "\n"))
}

var cheatMenuOptions = []string{
	"Max All Stats",
	"Min All Stats (Critical)",
	"Full Energy",
	"Empty Energy (Auto-sleep)",
	"Mood: Normal",
	"Mood: Playful",
	"Mood: Lazy",
	"Mood: Needy",
	"Toggle Illness",
	"Toggle Sleep",
	"Type: Early Bird 🌅",
	"Type: Normal ☀️",
	"Type: Night Owl 🦉",
	"Age +24h",
	"Kill Pet",
	"Back",
}

func (m Model) renderCheatMenu() string {
	var menuItems []string
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF0000")).
		Render("⚠️  CHEAT MENU ⚠️")

	for i, choice := range cheatMenuOptions {
		cursor := " "
		if m.CheatChoice == i {
			cursor = ">"
		}
		menuItems = append(menuItems, fmt.Sprintf("%s %s", cursor, choice))
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		gameStyles.menuBox.Render(strings.Join(menuItems, "\n")),
		"",
		gameStyles.status.Render("Press 'c' or Esc to exit"),
	)
}

func (m *Model) executeCheat() {
	switch m.CheatChoice {
	case 0: // Max All Stats
		m.modifyStats(func(p *pet.Pet) {
			p.Hunger = pet.MaxStat
			p.Happiness = pet.MaxStat
			p.Energy = pet.MaxStat
			p.Health = pet.MaxStat
		})
		m.setMessage("🎮 All stats maxed!")
	case 1: // Min All Stats (Critical)
		m.modifyStats(func(p *pet.Pet) {
			p.Hunger = 10
			p.Happiness = 10
			p.Energy = 10
			p.Health = 10
		})
		m.setMessage("🎮 Stats set to critical!")
	case 2: // Full Energy
		m.modifyStats(func(p *pet.Pet) {
			p.Energy = pet.MaxStat
		})
		m.setMessage("🎮 Energy maxed!")
	case 3: // Empty Energy (Auto-sleep)
		m.modifyStats(func(p *pet.Pet) {
			p.Energy = 0
		})
		m.setMessage("🎮 Energy emptied!")
	case 4: // Mood: Normal
		m.modifyStats(func(p *pet.Pet) {
			p.Mood = "normal"
			p.MoodExpiresAt = nil
		})
		m.setMessage("🎮 Mood set to normal")
	case 5: // Mood: Playful
		m.modifyStats(func(p *pet.Pet) {
			p.Mood = "playful"
			p.MoodExpiresAt = nil
		})
		m.setMessage("🎮 Mood set to playful")
	case 6: // Mood: Lazy
		m.modifyStats(func(p *pet.Pet) {
			p.Mood = "lazy"
			p.MoodExpiresAt = nil
		})
		m.setMessage("🎮 Mood set to lazy")
	case 7: // Mood: Needy
		m.modifyStats(func(p *pet.Pet) {
			p.Mood = "needy"
			p.MoodExpiresAt = nil
		})
		m.setMessage("🎮 Mood set to needy")
	case 8: // Toggle Illness
		m.modifyStats(func(p *pet.Pet) {
			p.Illness = !p.Illness
		})
		if m.Pet.Illness {
			m.setMessage("🎮 Illness: ON")
		} else {
			m.setMessage("🎮 Illness: OFF")
		}
	case 9: // Toggle Sleep
		m.modifyStats(func(p *pet.Pet) {
			p.Sleeping = !p.Sleeping
			p.AutoSleepTime = nil
			p.FractionalEnergy = 0
		})
		if m.Pet.Sleeping {
			m.setMessage("🎮 Pet is now sleeping")
		} else {
			m.setMessage("🎮 Pet is now awake")
		}
	case 10: // Type: Early Bird
		m.modifyStats(func(p *pet.Pet) {
			p.Chronotype = pet.ChronotypeEarlyBird
		})
		m.setMessage("🎮 Type: 🌅 Early Bird (5am-9pm)")
	case 11: // Type: Normal
		m.modifyStats(func(p *pet.Pet) {
			p.Chronotype = pet.ChronotypeNormal
		})
		m.setMessage("🎮 Type: ☀️ Normal (7am-11pm)")
	case 12: // Type: Night Owl
		m.modifyStats(func(p *pet.Pet) {
			p.Chronotype = pet.ChronotypeNightOwl
		})
		m.setMessage("🎮 Type: 🦉 Night Owl (10am-2am)")
	case 13: // Age +24h
		m.modifyStats(func(p *pet.Pet) {
			if len(p.Logs) > 0 {
				p.Logs[0].Time = p.Logs[0].Time.Add(-24 * time.Hour)
			}
		})
		m.setMessage(fmt.Sprintf("🎮 Age advanced! Now %dh", m.Pet.Age))
	case 14: // Kill Pet
		m.modifyStats(func(p *pet.Pet) {
			p.Dead = true
			p.CauseOfDeath = "Cheats"
		})
		m.setMessage("🎮 Pet has been killed")
		m.ShowingAdoptPrompt = true
	case 15: // Back
		m.InCheatMenu = false
	}
}

func (m Model) renderAnimation() string {
	frame := GetAnimationFrame(m.Animation)
	formEmoji := m.Pet.GetFormEmoji()
	title := gameStyles.title.Render(formEmoji + " " + m.Pet.Name + " " + formEmoji)

	animStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")).
		Bold(true).
		Padding(1, 2)

	var status string
	if m.Message != "" && pet.TimeNow().Before(m.MessageExpires) {
		status = gameStyles.status.Render(m.Message)
	}

	sections := []string{
		title,
		"",
		animStyle.Render(frame),
	}

	if status != "" {
		sections = append(sections, "", status)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) deadView() string {
	if m.ShowingAdoptPrompt {
		return lipgloss.JoinVertical(
			lipgloss.Center,
			gameStyles.title.Render("💀 "+m.Pet.Name+" 💀"),
			"",
			gameStyles.status.Render("Your pet has passed away..."),
			gameStyles.status.Render("Cause of death: "+m.Pet.CauseOfDeath),
			gameStyles.status.Render("They lived for "+fmt.Sprintf("%d hours", m.Pet.Age)),
			"",
			gameStyles.menuBox.Render("Would you like to adopt a new pet?"),
			"",
			gameStyles.status.Render("Press 'y' for yes, 'n' for no"),
		)
	}
	return lipgloss.JoinVertical(
		lipgloss.Center,
		gameStyles.title.Render("💀 "+m.Pet.Name+" 💀"),
		"",
		gameStyles.status.Render("Your pet has passed away..."),
		gameStyles.status.Render("It will be remembered forever."),
		"",
		gameStyles.status.Render("Press q to exit"),
	)
}
