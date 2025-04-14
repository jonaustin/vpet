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
	deathTimeThreshold = 4 * time.Hour // Shorter Tamagotchi-style timer
	healthDecreaseRate = 2             // Health loss per hour
	ageStageThresholds = 24            // Hours per life stage
	illnessChance      = 0.1           // 10% chance per hour when health <50
	medicineEffect     = 30            // Health restored by medicine
	minNaturalLifespan = 72            // Hours before natural death possible

	// Stat change rates (per hour)
	hungerDecreaseRate    = 5
	sleepingHungerRate    = 3 // 70% of normal rate
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
	pet      Pet
	choice   int
	quitting bool
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

func (m *model) discipline() {
	m.modifyStats(func(p *Pet) {
		p.Happiness = int(math.Max(float64(p.Happiness-10), float64(minStat)))
		p.Hunger = max(p.Hunger-5, minStat)
		log.Printf("Disciplined pet. Happiness is now %d, Hunger is now %d", p.Happiness, p.Hunger)
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
		if int(t.Minute())%60 == 0 {
			hungerRate := hungerDecreaseRate
			if p.Sleeping {
				hungerRate = sleepingHungerRate
			}
			p.Hunger = max(p.Hunger-hungerRate, minStat)
			log.Printf("Hunger decreased to %d", p.Hunger)
		}

		if !p.Sleeping {
			// Energy decreases when awake
			if int(t.Minute())%120 == 0 {
				p.Energy = max(p.Energy-energyDecreaseRate, minStat)
				log.Printf("Energy decreased to %d", p.Energy)
			}
		} else {
			// Sleeping recovers energy faster
			if int(t.Minute())%60 == 0 {
				p.Energy = min(p.Energy+energyRecoveryRate, maxStat)
				if p.Energy >= maxStat {
					p.Sleeping = false
				}
				log.Printf("Energy increased to %d", p.Energy)
			}
		}

		// Happiness affected by hunger and energy
		if p.Hunger < 30 || p.Energy < 30 {
			if int(t.Minute())%60 == 0 {
				p.Happiness = max(p.Happiness-2, 0)
				log.Printf("Happiness decreased to %d", p.Happiness)
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
	if testCfg != nil {
		return Pet{
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
	}
	now := timeNow() // Already UTC
	pet := Pet{
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
	pet.LastStatus = getStatus(pet)
	// Add initial log entry
	pet.Logs = []LogEntry{{
		Time:      now,
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
	hoursElapsed := int(elapsed.Hours())
	totalMinutes := int(elapsed.Minutes())
	log.Printf("total minutes: %d\n", totalMinutes)

	// Store current status before updates
	oldStatus := pet.LastStatus
	if oldStatus == "" {
		oldStatus = getStatus(pet)
	}

	// Update age and life stage
	pet.Age += hoursElapsed
	pet.LifeStage = min(pet.Age/ageStageThresholds, 2)

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
		if pet.Energy >= maxStat {
			pet.Sleeping = false
		}
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

	// Calculate health degradation
	healthLoss := hoursElapsed * healthDecreaseRate
	if pet.Health > 0 {
		pet.Health = max(pet.Health-healthLoss, 0)
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

		// Check for natural death from old age
		if pet.Age >= minNaturalLifespan && rand.Float64() < float64(pet.Age-minNaturalLifespan)/1000 {
			pet.Dead = true
			pet.CauseOfDeath = "Old Age"
		}
	} else {
		pet.CriticalStartTime = nil // Reset if recovered
	}

	pet.LastSaved = now
	return pet
}

func saveState(p *Pet) {
	now := timeNow()
	if len(p.Logs) > 0 {
		birthTime := p.Logs[0].Time
		p.Age = int(now.Sub(birthTime).Hours())
	}
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
		pet:    pet,
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
			if m.choice < 5 {
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
			case 4: // Discipline
				m.discipline()
			case 5: // Quit
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
	choices := []string{"Feed", "Play", "Sleep", "Medicine", "Discipline", "Quit"}
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
	if p.Sleeping {
		return "😴 Sleeping"
	}
	if p.Energy < 30 {
		return "😾 Tired"
	}
	if p.Hunger < 30 {
		return "🙀 Hungry"
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
