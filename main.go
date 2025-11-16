package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LogEntry represents a status change event
type LogEntry struct {
	Time      time.Time `json:"time"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
}

// Game constants
const (
	defaultPetName     = "Charm Pet"
	maxStat            = 100
	minStat            = 0
	lowStatThreshold   = 30
	deathTimeThreshold = 12 * time.Hour // Time in critical state before death
	healthDecreaseRate = 2              // Health loss per hour
	ageStageThresholds = 48             // Hours per life stage
	illnessChance      = 0.1            // 10% chance per hour when health <50
	medicineEffect     = 30             // Health restored by medicine
	minNaturalLifespan = 168            // Hours before natural death possible (~1 week)

	// Stat change rates (per hour)
	hungerDecreaseRate    = 5
	sleepingHungerRate    = 3 // 70% of normal rate
	energyDecreaseRate    = 5
	energyRecoveryRate    = 10
	happinessDecreaseRate = 2

	feedHungerIncrease    = 30
	feedHappinessIncrease = 10
	playHappinessIncrease = 30
	playEnergyDecrease    = 10
	playHungerDecrease    = 5
)

// Pet represents the virtual pet's state
type Pet struct {
	Name              string     `json:"name"`
	Hunger            int        `json:"hunger"`
	Happiness         int        `json:"happiness"`
	Energy            int        `json:"energy"`
	Health            int        `json:"health"` // New health metric
	Age               int        `json:"age"`    // In hours
	LifeStage         int        `json:"stage"`  // 0=baby, 1=child, 2=adult
	Sleeping          bool       `json:"sleeping"`
	Dead              bool       `json:"dead"`
	CauseOfDeath      string     `json:"cause_of_death,omitempty"`
	LastSaved         time.Time  `json:"last_saved"`
	CriticalStartTime *time.Time `json:"critical_start_time,omitempty"`
	Illness           bool       `json:"illness"` // Random sickness flag
	LastStatus        string     `json:"last_status,omitempty"`
	Logs              []LogEntry `json:"logs,omitempty"`
}

// model represents the game state
type model struct {
	pet                Pet
	choice             int
	quitting           bool
	showingAdoptPrompt bool
}

// Helper function to modify stats and save immediately
func (m *model) modifyStats(f func(*Pet)) {
	f(&m.pet)
	saveState(&m.pet)
}

// Pet state modification functions
func (m *model) administerMedicine() {
	m.modifyStats(func(p *Pet) {
		p.Illness = false
		p.Health = min(p.Health+medicineEffect, maxStat)
		log.Printf("Administered medicine. Health is now %d", p.Health)
	})
}

func (m *model) feed() {
	m.modifyStats(func(p *Pet) {
		p.Sleeping = false
		p.Hunger = min(p.Hunger+feedHungerIncrease, maxStat)
		p.Happiness = min(p.Happiness+feedHappinessIncrease, maxStat)
		log.Printf("Fed pet. Hunger is now %d, Happiness is now %d", p.Hunger, p.Happiness)
	})
}

// play increases happiness but decreases energy and hunger
func (m *model) play() {
	m.modifyStats(func(p *Pet) {
		p.Sleeping = false
		p.Happiness = min(p.Happiness+playHappinessIncrease, maxStat)
		p.Energy = max(p.Energy-playEnergyDecrease, minStat)
		p.Hunger = max(p.Hunger-playHungerDecrease, minStat)
		log.Printf("Played with pet. Happiness is now %d, Energy is now %d, Hunger is now %d", p.Happiness, p.Energy, p.Hunger)
	})
}

func (m *model) toggleSleep() {
	m.modifyStats(func(p *Pet) {
		p.Sleeping = !p.Sleeping
		log.Printf("Pet is now sleeping: %t", p.Sleeping)
	})
}

func (m *model) updateHourlyStats(t time.Time) {
	m.modifyStats(func(p *Pet) {
		// Hunger decreases every hour (reduced rate while sleeping)
		if int(t.Minute()) == 0 {
			hungerRate := hungerDecreaseRate
			if p.Sleeping {
				hungerRate = sleepingHungerRate
			}
			p.Hunger = max(p.Hunger-hungerRate, minStat)
			log.Printf("Hunger decreased to %d", p.Hunger)
		}

		if !p.Sleeping {
			// Energy decreases every 2 hours when awake
			if int(t.Hour())%2 == 0 && int(t.Minute()) == 0 {
				p.Energy = max(p.Energy-energyDecreaseRate, minStat)
				log.Printf("Energy decreased to %d", p.Energy)
			}
		} else {
			// Sleeping recovers energy faster
			if int(t.Minute()) == 0 {
				p.Energy = min(p.Energy+energyRecoveryRate, maxStat)
				log.Printf("Energy increased to %d", p.Energy)
			}
		}

		// Happiness affected by hunger and energy
		if p.Hunger < 30 || p.Energy < 30 {
			if int(t.Minute()) == 0 {
				p.Happiness = max(p.Happiness-2, 0)
				log.Printf("Happiness decreased to %d", p.Happiness)
			}
		}

		// Health decreases when any stat is critically low
		if p.Hunger < 15 || p.Happiness < 15 || p.Energy < 15 {
			if int(t.Minute()) == 0 { // Every hour
				healthRate := 2 // 2%/hr when awake
				if p.Sleeping {
					healthRate = 1 // 1%/hr when sleeping
				}
				p.Health = max(p.Health-healthRate, minStat)
				log.Printf("Health decreased to %d", p.Health)
			}
		}
	})
}

var (
	timeNow     = func() time.Time { return time.Now().UTC() } // Always use UTC time
	randFloat64 = rand.Float64                                 // Expose random function for testing

	gameStyles = struct {
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
)

// TestConfig allows overriding default values for testing
type TestConfig struct {
	InitialHunger    int
	InitialHappiness int
	InitialEnergy    int
	Health           int
	Age              int
	Illness          bool
	IsSleeping       bool
	LastSavedTime    time.Time
}

// newPet creates a new pet with default values or test values if provided
func newPet(testCfg *TestConfig) Pet {
	now := timeNow() // Already UTC
	var pet Pet
	var birthTime time.Time

	if testCfg != nil {
		birthTime = testCfg.LastSavedTime
		pet = Pet{
			Name:      defaultPetName,
			Hunger:    testCfg.InitialHunger,
			Happiness: testCfg.InitialHappiness,
			Energy:    testCfg.InitialEnergy,
			Health:    testCfg.Health,
			Age:       0,
			LifeStage: 0,
			Sleeping:  testCfg.IsSleeping,
			LastSaved: testCfg.LastSavedTime,
			Illness:   testCfg.Illness,
		}
	} else {
		birthTime = now
		pet = Pet{
			Name:      defaultPetName,
			Hunger:    maxStat,
			Happiness: maxStat,
			Energy:    maxStat,
			Health:    maxStat,
			Age:       0,
			LifeStage: 0,
			Sleeping:  false,
			LastSaved: now,
			Illness:   false,
		}
	}

	pet.LastStatus = getStatus(pet)
	// Add initial log entry with birth time
	pet.Logs = []LogEntry{{
		Time:      birthTime,
		OldStatus: "",
		NewStatus: pet.LastStatus,
	}}
	log.Printf("Created new pet: %s", pet.Name)
	return pet
}

// loadState loads the pet's state from file or creates a new pet
var testConfigPath string // Used for testing

func getConfigPath() string {
	if testConfigPath != "" {
		return testConfigPath
	}
	configDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	configPath := filepath.Join(configDir, ".config", "vpet", "pet.json")
	dirPath := filepath.Dir(configPath)

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		log.Printf("Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	return configPath
}

func loadState() Pet {
	configPath := getConfigPath()
	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Printf("Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Error reading state file: %v. Creating new pet.", err)
		return newPet(nil)
	}

	var pet Pet
	if err := json.Unmarshal(data, &pet); err != nil {
		log.Printf("Error loading state: %v. Creating new pet.", err)
		return newPet(nil)
	}

	// Update stats based on elapsed time and check for death
	now := timeNow()
	log.Printf("last saved: %s\n", pet.LastSaved.UTC())
	elapsed := now.Sub(pet.LastSaved.UTC()) // Ensure UTC comparison
	log.Printf("elapsed %f\n", elapsed.Seconds())
	totalMinutes := int(elapsed.Minutes())
	log.Printf("total minutes: %d\n", totalMinutes)

	// Store current status before updates
	oldStatus := pet.LastStatus
	if oldStatus == "" {
		oldStatus = getStatus(pet)
	}

	// Update age and life stage
	// Calculate age from birth time to avoid drift from integer truncation
	birthTime := pet.Logs[0].Time
	pet.Age = int(now.Sub(birthTime).Hours())

	// Calculate life stage based on age
	if pet.Age < ageStageThresholds {
		pet.LifeStage = 0 // Baby
	} else if pet.Age < 2*ageStageThresholds {
		pet.LifeStage = 1 // Child
	} else {
		pet.LifeStage = 2 // Adult
	}

	// Check death condition first
	if pet.Dead {
		return pet
	}

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
	}

	// Update happiness if stats are low
	if pet.Hunger < lowStatThreshold || pet.Energy < lowStatThreshold {
		happinessLoss := (totalMinutes / 60) * happinessDecreaseRate
		pet.Happiness = max(pet.Happiness-happinessLoss, minStat)
	}

	// Check for random illness when health is low
	if pet.Health < 50 && !pet.Illness {
		if randFloat64() < illnessChance {
			pet.Illness = true
		}
	} else if pet.Health >= 50 {
		// Clear illness when health returns to safe levels
		pet.Illness = false
	}

	// Health decreases when any stat is critically low
	if pet.Hunger < 15 || pet.Happiness < 15 || pet.Energy < 15 {
		healthRate := 2 // 2%/hr when awake
		if pet.Sleeping {
			healthRate = 1 // 1%/hr when sleeping
		}
		healthLoss := (totalMinutes / 60) * healthRate
		pet.Health = max(pet.Health-healthLoss, minStat)
	}

	// Check if any critical stat is below threshold
	inCriticalState := pet.Health <= 20 || pet.Hunger < 10 ||
		pet.Happiness < 10 || pet.Energy < 10

	// Track time in critical state
	if inCriticalState {
		if pet.CriticalStartTime == nil {
			pet.CriticalStartTime = &now
		}

		// Check if been critical too long
		if now.Sub(*pet.CriticalStartTime) > deathTimeThreshold {
			pet.Dead = true
			pet.CauseOfDeath = "Neglect"

			if pet.Hunger <= 0 {
				pet.CauseOfDeath = "Starvation"
			} else if pet.Illness {
				pet.CauseOfDeath = "Sickness"
			}
		}
	} else {
		pet.CriticalStartTime = nil // Reset if recovered
	}

	// Check for natural death from old age (independent of critical state)
	if pet.Age >= minNaturalLifespan && randFloat64() < float64(pet.Age-minNaturalLifespan)/1000 {
		pet.Dead = true
		pet.CauseOfDeath = "Old Age"
	}

	pet.LastSaved = now
	return pet
}

func saveState(p *Pet) {
	now := timeNow()
	// Calculate age from birth time
	birthTime := p.Logs[0].Time
	p.Age = int(now.Sub(birthTime).Hours())
	p.LastSaved = now

	// Add status change tracking
	currentStatus := getStatus(*p)
	if p.LastStatus == "" {
		p.LastStatus = currentStatus
	}

	if currentStatus != p.LastStatus {
		// Initialize logs array if needed
		if p.Logs == nil {
			p.Logs = []LogEntry{}
		}

		// Add new log entry using the already computed 'now'
		newLog := LogEntry{
			Time:      now,
			OldStatus: p.LastStatus,
			NewStatus: currentStatus,
		}
		p.Logs = append(p.Logs, newLog)
		p.LastStatus = currentStatus
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		log.Printf("Error saving state: %v", err)
		return
	}
	if err := os.WriteFile(getConfigPath(), data, 0644); err != nil {
		log.Printf("Error writing state: %v", err)
	}
}

func initialModel(testCfg *TestConfig) model {
	var pet Pet
	if testCfg != nil {
		pet = newPet(testCfg)
	} else {
		pet = loadState()
	}
	return model{
		pet:                pet,
		choice:             0,
		showingAdoptPrompt: pet.Dead, // Show adoption prompt if pet is already dead
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
		case "y":
			// Handle adoption prompt
			if m.pet.Dead && m.showingAdoptPrompt {
				// Create new pet
				m.pet = newPet(nil)
				m.showingAdoptPrompt = false
				m.choice = 0
				saveState(&m.pet)
				return m, nil
			}
		case "n":
			// Handle adoption prompt rejection
			if m.pet.Dead && m.showingAdoptPrompt {
				m.showingAdoptPrompt = false
				return m, nil
			}
		case "up", "k":
			if m.choice > 0 {
				m.choice--
			}
		case "down", "j":
			if m.choice < 4 {
				m.choice++
			}
		case "enter", " ":
			if m.pet.Dead {
				return m, nil // Ignore input when dead
			}
			switch m.choice {
			case 0: // Feed
				m.feed()
			case 1: // Play
				m.play()
			case 2: // Sleep
				m.toggleSleep()
			case 3: // Medicine
				m.administerMedicine()
			case 4: // Quit
				m.quitting = true
				return m, tea.Quit
			}
		}

	case tickMsg:
		m.updateHourlyStats(time.Time(msg))
		// If pet just died, show adoption prompt
		if m.pet.Dead && !m.showingAdoptPrompt {
			m.showingAdoptPrompt = true
		}
		return m, tick()
	}

	return m, nil
}

func (m model) View() string {
	if m.pet.Dead {
		return m.deadView()
	}
	if m.quitting {
		return "Thanks for playing!\n"
	}

	// Build the view from components
	title := gameStyles.title.Render("😺 " + m.pet.Name + " 😺")
	stats := m.renderStats()
	status := m.renderStatus()
	menu := m.renderMenu()

	// Join all sections vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		stats,
		"",
		status,
		"",
		menu,
		"",
		gameStyles.status.Render("Use arrows to move • enter to select • q to quit"),
	)
}

func (m model) renderStats() string {
	lifeStage := "Baby"
	switch m.pet.LifeStage {
	case 1:
		lifeStage = "Child"
	case 2:
		lifeStage = "Adult"
	}

	stats := []struct {
		name, value string
	}{
		{"Hunger", fmt.Sprintf("%d%%", m.pet.Hunger)},
		{"Happiness", fmt.Sprintf("%d%%", m.pet.Happiness)},
		{"Energy", fmt.Sprintf("%d%%", m.pet.Energy)},
		{"Health", fmt.Sprintf("%d%%", m.pet.Health)},
		{"Age", fmt.Sprintf("%dh", m.pet.Age)},
		{"Illness", map[bool]string{true: "Yes", false: "No"}[m.pet.Illness]},
		{"Life Stage", lifeStage},
	}

	var lines []string
	for _, stat := range stats {
		lines = append(lines, fmt.Sprintf("%-10s %s", stat.name+":", stat.value))
	}

	return gameStyles.stats.Render(strings.Join(lines, "\n"))
}

func (m model) renderStatus() string {
	return gameStyles.status.Render(fmt.Sprintf("Status: %s", getStatus(m.pet)))
}

func (m model) renderMenu() string {
	choices := []string{"Feed", "Play", "Sleep", "Medicine", "Quit"}
	var menuItems []string

	for i, choice := range choices {
		cursor := " "
		if m.choice == i {
			cursor = ">"
		}
		menuItems = append(menuItems, fmt.Sprintf("%s %s", cursor, choice))
	}

	return gameStyles.menuBox.Render(strings.Join(menuItems, "\n"))
}

func (m model) deadView() string {
	if m.showingAdoptPrompt {
		return lipgloss.JoinVertical(
			lipgloss.Center,
			gameStyles.title.Render("💀 "+m.pet.Name+" 💀"),
			"",
			gameStyles.status.Render("Your pet has passed away..."),
			gameStyles.status.Render("Cause of death: "+m.pet.CauseOfDeath),
			gameStyles.status.Render("They lived for "+fmt.Sprintf("%d hours", m.pet.Age)),
			"",
			gameStyles.menuBox.Render("Would you like to adopt a new pet?"),
			"",
			gameStyles.status.Render("Press 'y' for yes, 'n' for no"),
		)
	}
	return lipgloss.JoinVertical(
		lipgloss.Center,
		gameStyles.title.Render("💀 "+m.pet.Name+" 💀"),
		"",
		gameStyles.status.Render("Your pet has passed away..."),
		gameStyles.status.Render("It will be remembered forever."),
		"",
		gameStyles.status.Render("Press q to exit"),
	)
}

func getStatus(p Pet) string {
	if p.Dead {
		return "💀 Dead"
	}

	// Find the lowest stat to prioritize critical issues
	lowestStat := p.Health
	lowestStatus := "🤢 Sick"

	if p.Energy < lowestStat {
		lowestStat = p.Energy
		lowestStatus = "😾 Tired"
	}
	if p.Hunger < lowestStat {
		lowestStat = p.Hunger
		lowestStatus = "🙀 Hungry"
	}
	if p.Happiness < lowestStat {
		lowestStat = p.Happiness
		lowestStatus = "😿 Sad"
	}

	// Show critical issues even when sleeping
	if lowestStat < 30 {
		return lowestStatus
	}

	// Only show sleeping if no critical issues
	if p.Sleeping {
		return "😴 Sleeping"
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
	// Configure logging to write to ./vpet.log
	logFile := "./vpet.log"
	logFileHandle, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer logFileHandle.Close()
	log.SetOutput(logFileHandle)

	updateOnly := flag.Bool("u", false, "Update pet stats only, don't run UI")
	statusFlag := flag.Bool("status", false, "Output current status emoji")
	flag.Parse()

	if *statusFlag {
		pet := loadState()
		fmt.Print(strings.Split(getStatus(pet), " ")[0])
		return
	}

	if *updateOnly {
		pet := loadState() // This already updates based on elapsed time
		saveState(&pet)    // Save the updated stats
		return
	}

	p := tea.NewProgram(initialModel(nil))
	if _, err := p.Run(); err != nil {
		log.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
