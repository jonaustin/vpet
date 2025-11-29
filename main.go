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

	// Care quality thresholds for evolution
	perfectCareThreshold = 85
	goodCareThreshold    = 70
	poorCareThreshold    = 40
	neglectThreshold     = 20

	// Autonomous behavior thresholds
	autoSleepThreshold   = 20 // Energy level that triggers auto-sleep
	drowsyThreshold      = 40 // Energy level that shows drowsy status
	autoWakeEnergy       = 80 // Energy level to wake up automatically
	minSleepDuration     = 6  // Minimum hours of auto-sleep
	maxSleepDuration     = 8  // Maximum hours before forced wake
	hungryThreshold      = 30 // Hunger level to show "wants food"
	boredThreshold       = 30 // Happiness level to show "wants play"

	// Chronotype multipliers
	outsideActiveEnergyMult    = 1.5 // 50% faster energy drain outside active hours
	outsideActiveHappinessMult = 0.7 // 30% less happiness gain outside active hours
	preferredSleepRecoveryMult = 1.2 // 20% better sleep recovery during preferred hours

	// Bonding system constants
	maxBond                = 100          // Maximum bond level
	initialBond            = 50           // Starting bond for new pets
	bondDecayThreshold     = 24           // Hours before bond starts decaying from neglect
	bondDecayRate          = 1            // Bond lost per 12 hours of neglect beyond threshold
	spamPreventionWindow   = 1 * time.Hour // Time window to check for repeated actions
	minBondMultiplier      = 0.5          // Action effectiveness at 0 bond
	maxBondMultiplier      = 1.0          // Action effectiveness at 100 bond
	bondGainWellTimed      = 2            // Bond gained for well-timed action
	bondGainNormal         = 1            // Bond gained for normal action
	illnessResistanceBond  = 70           // Bond level that starts reducing illness chance
	maxInteractionHistory  = 20           // Keep last 20 interactions
)

// Chronotype constants
const (
	ChronotypeEarlyBird = "early_bird" // 5am-9pm active
	ChronotypeNormal    = "normal"     // 7am-11pm active
	ChronotypeNightOwl  = "night_owl"  // 10am-2am active
)

// Pet forms for evolution system
const (
	FormBaby PetForm = iota
	// Child forms
	FormHealthyChild
	FormTroubledChild
	FormSicklyChild
	// Adult forms
	FormEliteAdult
	FormStandardAdult
	FormGrumpyAdult
	FormRedeemedAdult
	FormDelinquentAdult
	FormWeakAdult
)

type PetForm int

// Event types for the life events system
type EventType string

const (
	EventNone           EventType = ""
	EventChasing        EventType = "chasing"        // Chasing butterfly/bug
	EventFoundSomething EventType = "found"          // Found a mystery item
	EventScared         EventType = "scared"         // Scared of something
	EventDaydreaming    EventType = "daydreaming"    // Lost in thought
	EventAteSomething   EventType = "ate_something"  // Ate something questionable
	EventSinging        EventType = "singing"        // Rare happy moment
	EventNightmare      EventType = "nightmare"      // Bad dream while sleeping
	EventLearnedTrick   EventType = "learned_trick"  // Achievement unlocked
	EventZoomies        EventType = "zoomies"        // Sudden burst of energy
	EventCuddles        EventType = "cuddles"        // Wants affection
)

// Event represents a life event happening to the pet
type Event struct {
	Type      EventType `json:"type"`
	StartTime time.Time `json:"start_time"`
	ExpiresAt time.Time `json:"expires_at"`
	Responded bool      `json:"responded"` // Player has responded to this event
}

// Trait represents a personality characteristic that affects pet behavior
type Trait struct {
	Name      string             `json:"name"`
	Category  string             `json:"category"`  // "temperament", "appetite", "sociability", "constitution"
	Modifiers map[string]float64 `json:"modifiers"` // stat_name -> multiplier (e.g., "hunger_decay": 1.2)
}

// Interaction represents a player action with the pet
type Interaction struct {
	Type string    `json:"type"` // "feed", "play", "medicine"
	Time time.Time `json:"time"`
}

// EventLogEntry records past events for the pet's "memory"
type EventLogEntry struct {
	Type       EventType `json:"type"`
	Time       time.Time `json:"time"`
	WasIgnored bool      `json:"was_ignored"`
}

// CareQuality tracks average stats during a life stage
type CareQuality struct {
	AvgHunger    int `json:"avg_hunger"`
	AvgHappiness int `json:"avg_happiness"`
	AvgEnergy    int `json:"avg_energy"`
	AvgHealth    int `json:"avg_health"`
}

// Pet represents the virtual pet's state
type Pet struct {
	Name               string                 `json:"name"`
	Hunger             int                    `json:"hunger"`
	Happiness          int                    `json:"happiness"`
	Energy             int                    `json:"energy"`
	Health             int                    `json:"health"` // New health metric
	Age                int                    `json:"age"`    // In hours
	LifeStage          int                    `json:"stage"`  // 0=baby, 1=child, 2=adult
	Form               PetForm                `json:"form"`   // Current evolution form
	Sleeping           bool                   `json:"sleeping"`
	Dead               bool                   `json:"dead"`
	CauseOfDeath       string                 `json:"cause_of_death,omitempty"`
	LastSaved          time.Time              `json:"last_saved"`
	CriticalStartTime  *time.Time             `json:"critical_start_time,omitempty"`
	Illness            bool                   `json:"illness"` // Random sickness flag
	LastStatus         string                 `json:"last_status,omitempty"`
	Logs               []LogEntry             `json:"logs,omitempty"`
	CareQualityHistory map[int]CareQuality    `json:"care_quality_history,omitempty"` // Tracks care per stage
	StatCheckpoints    map[string][]StatCheck `json:"stat_checkpoints,omitempty"`     // For calculating averages

	// Autonomous behavior fields
	Mood          string     `json:"mood,omitempty"`            // "normal", "playful", "lazy", "needy"
	MoodExpiresAt *time.Time `json:"mood_expires_at,omitempty"` // When current mood ends
	AutoSleepTime *time.Time `json:"auto_sleep_time,omitempty"` // When pet fell asleep on its own

	// Life events system
	CurrentEvent *Event          `json:"current_event,omitempty"` // Active event, if any
	EventLog     []EventLogEntry `json:"event_log,omitempty"`     // History of past events

	// Circadian rhythm
	Chronotype string `json:"chronotype,omitempty"` // "early_bird", "normal", "night_owl"

	// Personality traits
	Traits []Trait `json:"traits,omitempty"` // Personality characteristics affecting behavior

	// Bonding system
	Bond             int           `json:"bond,omitempty"`              // 0-100 relationship quality
	LastInteractions []Interaction `json:"last_interactions,omitempty"` // Recent actions for spam detection

	// Fractional stat accumulators (to handle frequent small updates)
	FractionalEnergy float64 `json:"fractional_energy,omitempty"` // Accumulated fractional energy gain
}

// StatCheck records stats at a point in time for averaging
type StatCheck struct {
	Time      time.Time `json:"time"`
	Hunger    int       `json:"hunger"`
	Happiness int       `json:"happiness"`
	Energy    int       `json:"energy"`
	Health    int       `json:"health"`
}

// model represents the game state
type model struct {
	pet                Pet
	choice             int
	quitting           bool
	showingAdoptPrompt bool
	evolutionMessage   string
	message            string    // Feedback message (refusal, action result, etc.)
	messageExpires     time.Time // When to clear the message
	inCheatMenu        bool      // Hidden cheat menu active
	cheatChoice        int       // Selected cheat option
}

// Evolution helper functions
func (p *Pet) recordStatCheckpoint() {
	if p.StatCheckpoints == nil {
		p.StatCheckpoints = make(map[string][]StatCheck)
	}

	stageKey := fmt.Sprintf("stage_%d", p.LifeStage)
	checkpoint := StatCheck{
		Time:      timeNow(),
		Hunger:    p.Hunger,
		Happiness: p.Happiness,
		Energy:    p.Energy,
		Health:    p.Health,
	}

	p.StatCheckpoints[stageKey] = append(p.StatCheckpoints[stageKey], checkpoint)
}

func (p *Pet) calculateCareQuality(stage int) CareQuality {
	stageKey := fmt.Sprintf("stage_%d", stage)
	checkpoints := p.StatCheckpoints[stageKey]

	if len(checkpoints) == 0 {
		// No data, assume perfect care
		return CareQuality{
			AvgHunger:    maxStat,
			AvgHappiness: maxStat,
			AvgEnergy:    maxStat,
			AvgHealth:    maxStat,
		}
	}

	var totalHunger, totalHappiness, totalEnergy, totalHealth int
	for _, checkpoint := range checkpoints {
		totalHunger += checkpoint.Hunger
		totalHappiness += checkpoint.Happiness
		totalEnergy += checkpoint.Energy
		totalHealth += checkpoint.Health
	}

	count := len(checkpoints)
	return CareQuality{
		AvgHunger:    totalHunger / count,
		AvgHappiness: totalHappiness / count,
		AvgEnergy:    totalEnergy / count,
		AvgHealth:    totalHealth / count,
	}
}

func (cq CareQuality) overallAverage() int {
	return (cq.AvgHunger + cq.AvgHappiness + cq.AvgEnergy + cq.AvgHealth) / 4
}

func (p *Pet) evolve(newStage int) {
	// Calculate care quality from previous stage
	prevStage := newStage - 1
	careQuality := p.calculateCareQuality(prevStage)

	// Store care quality history
	if p.CareQualityHistory == nil {
		p.CareQualityHistory = make(map[int]CareQuality)
	}
	p.CareQualityHistory[prevStage] = careQuality

	// Determine new form based on stage and care quality
	avgCare := careQuality.overallAverage()

	switch newStage {
	case 1: // Evolving to Child
		if avgCare >= goodCareThreshold {
			p.Form = FormHealthyChild
		} else if avgCare >= poorCareThreshold {
			p.Form = FormTroubledChild
		} else {
			p.Form = FormSicklyChild
		}

	case 2: // Evolving to Adult
		switch p.Form {
		case FormHealthyChild:
			if avgCare >= perfectCareThreshold {
				p.Form = FormEliteAdult
			} else if avgCare >= goodCareThreshold {
				p.Form = FormStandardAdult
			} else {
				p.Form = FormGrumpyAdult
			}
		case FormTroubledChild:
			if avgCare >= goodCareThreshold {
				p.Form = FormRedeemedAdult
			} else {
				p.Form = FormDelinquentAdult
			}
		case FormSicklyChild:
			p.Form = FormWeakAdult
		}
	}

	log.Printf("Pet evolved to %s (care quality: %d%%)", p.getFormName(), avgCare)
}

func (p *Pet) getFormName() string {
	switch p.Form {
	case FormBaby:
		return "Baby"
	case FormHealthyChild:
		return "Healthy Child"
	case FormTroubledChild:
		return "Troubled Child"
	case FormSicklyChild:
		return "Sickly Child"
	case FormEliteAdult:
		return "Elite Adult"
	case FormStandardAdult:
		return "Standard Adult"
	case FormGrumpyAdult:
		return "Grumpy Adult"
	case FormRedeemedAdult:
		return "Redeemed Adult"
	case FormDelinquentAdult:
		return "Delinquent Adult"
	case FormWeakAdult:
		return "Weak Adult"
	default:
		return "Unknown"
	}
}

func (p *Pet) getFormEmoji() string {
	switch p.Form {
	case FormBaby:
		return "🐣"
	case FormHealthyChild:
		return "😊"
	case FormTroubledChild:
		return "😟"
	case FormSicklyChild:
		return "🤒"
	case FormEliteAdult:
		return "⭐"
	case FormStandardAdult:
		return "😺"
	case FormGrumpyAdult:
		return "😼"
	case FormRedeemedAdult:
		return "😸"
	case FormDelinquentAdult:
		return "😾"
	case FormWeakAdult:
		return "🤕"
	default:
		return "❓"
	}
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
		// Bond multiplier affects medicine effectiveness
		bondMultiplier := p.getBondMultiplier()
		healthGain := int(float64(medicineEffect) * bondMultiplier)
		p.Health = min(p.Health+healthGain, maxStat)

		// Record interaction and increase bond for caring for sick pet
		p.addInteraction("medicine")
		p.updateBond(bondGainWellTimed) // Always well-timed when pet is sick

		log.Printf("Administered medicine (bond mult: %.2f). Health is now %d", bondMultiplier, p.Health)
	})
}

func (m *model) setMessage(msg string) {
	m.message = msg
	m.messageExpires = timeNow().Add(3 * time.Second)
}

func (m *model) feed() {
	// Check if pet is too full
	if m.pet.Hunger >= 90 {
		m.setMessage("🍽️ Not hungry right now!")
		return
	}

	// Check for spam feeding
	recentFeeds := countRecentInteractions(m.pet.LastInteractions, "feed", spamPreventionWindow)

	m.modifyStats(func(p *Pet) {
		p.Sleeping = false
		// Clear auto-sleep time when woken by feeding
		p.AutoSleepTime = nil
		p.FractionalEnergy = 0 // Reset fractional accumulator when waking

		// Calculate effectiveness with diminishing returns for spam
		effectiveness := 1.0
		if recentFeeds > 0 {
			effectiveness = 1.0 / float64(recentFeeds+1) // 1.0, 0.5, 0.33, 0.25...
		}

		// Apply bond multiplier
		bondMultiplier := p.getBondMultiplier()

		// Apply trait modifiers to feeding
		hungerGain := int(float64(feedHungerIncrease) * p.getTraitModifier("feed_bonus") * effectiveness * bondMultiplier)
		happinessGain := int(float64(feedHappinessIncrease) * p.getTraitModifier("feed_bonus_happiness") * effectiveness * bondMultiplier)

		p.Hunger = min(p.Hunger+hungerGain, maxStat)
		p.Happiness = min(p.Happiness+happinessGain, maxStat)

		// Record interaction
		p.addInteraction("feed")

		// Update bond
		if recentFeeds == 0 && p.Hunger < 50 {
			// Well-timed feeding (not spam, pet was hungry)
			p.updateBond(bondGainWellTimed)
		} else if recentFeeds == 0 {
			// Normal feeding (not spam, but pet wasn't very hungry)
			p.updateBond(bondGainNormal)
		}
		// No bond gain for spam feeding

		log.Printf("Fed pet (effectiveness: %.2f, bond mult: %.2f). Hunger is now %d, Happiness is now %d",
			effectiveness, bondMultiplier, p.Hunger, p.Happiness)
	})
	m.setMessage("🍖 Yum!")
}

// play increases happiness but decreases energy and hunger
func (m *model) play() {
	// Check if pet is too tired to play
	if m.pet.Energy < autoSleepThreshold {
		m.setMessage("😴 Too tired to play...")
		return
	}

	// Check if pet is in lazy mood
	if m.pet.Mood == "lazy" && m.pet.Energy < 50 {
		m.setMessage("😪 Not in the mood to play...")
		return
	}

	// Check if it's active hours for happiness bonus (use local time)
	// Use timeNow() which is mocked in tests
	currentHour := timeNow().Local().Hour()
	isActive := isActiveHours(&m.pet, currentHour)

	// Check for spam playing
	recentPlays := countRecentInteractions(m.pet.LastInteractions, "play", spamPreventionWindow)

	m.modifyStats(func(p *Pet) {
		p.Sleeping = false
		// Clear auto-sleep time when woken by playing
		p.AutoSleepTime = nil
		p.FractionalEnergy = 0 // Reset fractional accumulator when waking

		// Calculate effectiveness with diminishing returns for spam
		effectiveness := 1.0
		if recentPlays > 0 {
			effectiveness = 1.0 / float64(recentPlays+1)
		}

		// Apply bond multiplier
		bondMultiplier := p.getBondMultiplier()

		// Apply happiness gain with chronotype, trait, bond, and spam multipliers
		happinessGain := float64(playHappinessIncrease)
		if !isActive {
			// Reduced happiness outside active hours
			happinessGain *= outsideActiveHappinessMult
		}
		// Apply trait modifier for play bonus
		happinessGain *= p.getTraitModifier("play_bonus")
		// Apply bond and effectiveness multipliers
		happinessGain *= bondMultiplier * effectiveness

		p.Happiness = min(p.Happiness+int(happinessGain), maxStat)
		p.Energy = max(p.Energy-playEnergyDecrease, minStat)
		p.Hunger = max(p.Hunger-playHungerDecrease, minStat)

		// Record interaction
		p.addInteraction("play")

		// Update bond
		if recentPlays == 0 && p.Happiness < 50 {
			// Well-timed play (not spam, pet was bored)
			p.updateBond(bondGainWellTimed)
		} else if recentPlays == 0 {
			// Normal play (not spam, but pet wasn't very bored)
			p.updateBond(bondGainNormal)
		}
		// No bond gain for spam playing

		log.Printf("Played with pet (effectiveness: %.2f, bond mult: %.2f). Happiness is now %d, Energy is now %d, Hunger is now %d",
			effectiveness, bondMultiplier, p.Happiness, p.Energy, p.Hunger)
	})

	// Different messages based on mood and time
	if !isActive {
		m.setMessage("🥱 *yawn* ...play time...")
	} else if m.pet.Mood == "playful" {
		m.setMessage("🎉 So much fun!")
	} else {
		m.setMessage("🎾 Wheee!")
	}
}

func (m *model) toggleSleep() {
	m.modifyStats(func(p *Pet) {
		p.Sleeping = !p.Sleeping
		// Clear auto-sleep time if manually toggled
		p.AutoSleepTime = nil
		// Reset fractional energy when toggling sleep state
		p.FractionalEnergy = 0
		log.Printf("Pet is now sleeping: %t", p.Sleeping)
	})
}

// applyAutonomousBehavior makes the pet act on its own based on current state
func applyAutonomousBehavior(p *Pet) {
	now := timeNow()
	// Use local time for chronotype checks (user's actual hour)
	// In tests, timeNow() is mocked, so we use its local hour
	currentHour := now.Local().Hour()
	isActive := isActiveHours(p, currentHour)

	// Determine auto-sleep threshold based on chronotype
	sleepThreshold := autoSleepThreshold
	if !isActive {
		// Outside active hours, pet gets sleepy at higher energy (drowsy threshold)
		sleepThreshold = drowsyThreshold
	}

	// Auto-sleep when exhausted (or drowsy outside active hours)
	if p.Energy <= sleepThreshold && !p.Sleeping && !p.Dead {
		p.Sleeping = true
		p.AutoSleepTime = &now
		if isActive {
			log.Printf("Pet fell asleep automatically due to low energy (%d)", p.Energy)
		} else {
			log.Printf("Pet fell asleep (outside active hours, energy: %d)", p.Energy)
		}
	}

	// Auto-wake after sufficient sleep and energy restored
	if p.Sleeping && p.AutoSleepTime != nil {
		sleepDuration := now.Sub(*p.AutoSleepTime)
		sleepHours := sleepDuration.Hours()

		// Wake up conditions:
		// 1. Slept minimum hours AND energy restored AND it's active hours, OR
		// 2. Slept maximum hours (force wake regardless)
		shouldWake := false
		if sleepHours >= maxSleepDuration {
			shouldWake = true
		} else if sleepHours >= minSleepDuration && p.Energy >= autoWakeEnergy && isActive {
			shouldWake = true
		}

		if shouldWake {
			p.Sleeping = false
			p.AutoSleepTime = nil
			p.FractionalEnergy = 0 // Reset fractional accumulator when waking
			log.Printf("Pet woke up automatically after %.1f hours of sleep (Energy: %d)", sleepHours, p.Energy)
		}
	}

	// Random mood changes
	if p.Mood == "" {
		p.Mood = "normal"
	}
	if p.MoodExpiresAt == nil || now.After(*p.MoodExpiresAt) {
		// Mood influenced by current stats
		var newMood string
		roll := randFloat64()

		if p.Energy < drowsyThreshold {
			// Tired pets are more likely to be lazy
			if roll < 0.6 {
				newMood = "lazy"
			} else if roll < 0.8 {
				newMood = "needy"
			} else {
				newMood = "normal"
			}
		} else if p.Happiness < boredThreshold {
			// Unhappy pets want attention
			if roll < 0.5 {
				newMood = "needy"
			} else if roll < 0.7 {
				newMood = "playful"
			} else {
				newMood = "normal"
			}
		} else if p.Hunger < hungryThreshold {
			// Hungry pets are needy
			if roll < 0.5 {
				newMood = "needy"
			} else {
				newMood = "normal"
			}
		} else {
			// Happy, fed, rested pet - random mood
			if roll < 0.6 {
				newMood = "normal"
			} else if roll < 0.8 {
				newMood = "playful"
			} else if roll < 0.9 {
				newMood = "lazy"
			} else {
				newMood = "needy"
			}
		}

		p.Mood = newMood
		// Mood lasts 2-4 hours
		moodDuration := time.Duration(2+int(randFloat64()*2)) * time.Hour
		expires := now.Add(moodDuration)
		p.MoodExpiresAt = &expires
		log.Printf("Pet mood changed to: %s (expires in %.1f hours)", newMood, moodDuration.Hours())
	}
}

// Chronotype schedule: returns (wakeHour, sleepHour)
func getChronotypeSchedule(chronotype string) (int, int) {
	switch chronotype {
	case ChronotypeEarlyBird:
		return 5, 21 // 5am - 9pm
	case ChronotypeNightOwl:
		return 10, 2 // 10am - 2am (next day)
	default: // ChronotypeNormal
		return 7, 23 // 7am - 11pm
	}
}

// isActiveHours checks if the given hour is within the pet's active window
func isActiveHours(p *Pet, hour int) bool {
	wakeHour, sleepHour := getChronotypeSchedule(p.Chronotype)

	if sleepHour > wakeHour {
		// Normal case: wake and sleep on same day (e.g., 7am-11pm)
		return hour >= wakeHour && hour < sleepHour
	}
	// Night owl case: sleep hour is next day (e.g., 10am-2am)
	return hour >= wakeHour || hour < sleepHour
}

// getChronotypeName returns a display-friendly name
func getChronotypeName(chronotype string) string {
	switch chronotype {
	case ChronotypeEarlyBird:
		return "Early Bird"
	case ChronotypeNightOwl:
		return "Night Owl"
	default:
		return "Normal"
	}
}

// getChronotypeEmoji returns an emoji for the chronotype
func getChronotypeEmoji(chronotype string) string {
	switch chronotype {
	case ChronotypeEarlyBird:
		return "🌅"
	case ChronotypeNightOwl:
		return "🦉"
	default:
		return "☀️"
	}
}

// assignRandomChronotype picks a random chronotype for a new pet
func assignRandomChronotype() string {
	roll := randFloat64()
	if roll < 0.33 {
		return ChronotypeEarlyBird
	} else if roll < 0.66 {
		return ChronotypeNormal
	}
	return ChronotypeNightOwl
}

// generateTraits assigns random personality traits at birth
func generateTraits() []Trait {
	traitDefinitions := map[string][]Trait{
		"temperament": {
			{
				Name:     "Calm",
				Category: "temperament",
				Modifiers: map[string]float64{
					"energy_decay":    0.8,  // 20% slower energy drain
					"happiness_decay": 0.85, // 15% slower happiness decay
				},
			},
			{
				Name:     "Hyperactive",
				Category: "temperament",
				Modifiers: map[string]float64{
					"energy_decay": 1.3,  // 30% faster energy drain
					"play_bonus":   1.25, // 25% more happiness from play
				},
			},
		},
		"appetite": {
			{
				Name:     "Picky",
				Category: "appetite",
				Modifiers: map[string]float64{
					"feed_bonus": 0.75, // 25% less hunger gain from feeding
				},
			},
			{
				Name:     "Hungry",
				Category: "appetite",
				Modifiers: map[string]float64{
					"hunger_decay": 1.2,  // 20% faster hunger decay
					"feed_bonus":   1.25, // 25% more hunger gain from feeding
				},
			},
		},
		"sociability": {
			{
				Name:     "Independent",
				Category: "sociability",
				Modifiers: map[string]float64{
					"happiness_decay": 0.75, // 25% slower happiness decay
				},
			},
			{
				Name:     "Needy",
				Category: "sociability",
				Modifiers: map[string]float64{
					"happiness_decay": 1.15, // 15% faster happiness decay
					"play_bonus":      1.2,  // 20% more happiness from play
					"feed_bonus_happiness": 1.3, // 30% more happiness from feeding
				},
			},
		},
		"constitution": {
			{
				Name:     "Robust",
				Category: "constitution",
				Modifiers: map[string]float64{
					"illness_chance": 0.5, // 50% less likely to get sick
					"health_decay":   0.85, // 15% slower health decay
				},
			},
			{
				Name:     "Fragile",
				Category: "constitution",
				Modifiers: map[string]float64{
					"illness_chance": 1.8, // 80% more likely to get sick
					"health_decay":   1.2, // 20% faster health decay
				},
			},
		},
	}

	var traits []Trait
	for _, options := range traitDefinitions {
		// Pick a random trait from each category
		// Use modulo to ensure index is always valid (handles edge case where randFloat64() returns 1.0)
		index := int(randFloat64() * float64(len(options)))
		if index >= len(options) {
			index = len(options) - 1
		}
		selectedTrait := options[index]
		traits = append(traits, selectedTrait)
		log.Printf("Assigned %s trait: %s", selectedTrait.Category, selectedTrait.Name)
	}

	return traits
}

// getTraitModifier returns the combined modifier for a given stat type
func (p *Pet) getTraitModifier(modifierKey string) float64 {
	multiplier := 1.0
	for _, trait := range p.Traits {
		if mod, exists := trait.Modifiers[modifierKey]; exists {
			multiplier *= mod
		}
	}
	return multiplier
}

// countRecentInteractions counts how many times an action occurred within a time window
func countRecentInteractions(interactions []Interaction, actionType string, window time.Duration) int {
	now := timeNow()
	count := 0
	for _, interaction := range interactions {
		if interaction.Type == actionType && now.Sub(interaction.Time) < window {
			count++
		}
	}
	return count
}

// getBondMultiplier returns effectiveness multiplier based on bond level (0.5 to 1.0)
func (p *Pet) getBondMultiplier() float64 {
	// Linear scaling from minBondMultiplier at 0 bond to maxBondMultiplier at 100 bond
	return minBondMultiplier + (float64(p.Bond)/float64(maxBond))*(maxBondMultiplier-minBondMultiplier)
}

// addInteraction records an interaction and maintains history limit
func (p *Pet) addInteraction(actionType string) {
	p.LastInteractions = append(p.LastInteractions, Interaction{
		Type: actionType,
		Time: timeNow(),
	})
	// Keep only recent interactions
	if len(p.LastInteractions) > maxInteractionHistory {
		p.LastInteractions = p.LastInteractions[len(p.LastInteractions)-maxInteractionHistory:]
	}
}

// updateBond modifies bond level and clamps to valid range
func (p *Pet) updateBond(change int) {
	p.Bond = max(0, min(p.Bond+change, maxBond))
	log.Printf("Bond changed by %d, now %d", change, p.Bond)
}

// getBondDescription returns a descriptive label for the bond level
func getBondDescription(bond int) string {
	switch {
	case bond >= 90:
		return "💕 Soulmates"
	case bond >= 75:
		return "❤️ Best Friends"
	case bond >= 60:
		return "💛 Close"
	case bond >= 45:
		return "💚 Friendly"
	case bond >= 30:
		return "💙 Acquaintances"
	case bond >= 15:
		return "🤍 Distant"
	default:
		return "💔 Estranged"
	}
}

// eventDefinition describes an event's properties and conditions
type eventDefinition struct {
	Type        EventType
	Emoji       string
	Message     string
	Duration    time.Duration
	Condition   func(p *Pet) bool   // When can this event trigger?
	OnIgnored   func(p *Pet)        // What happens if ignored?
	OnResponded func(p *Pet) string // What happens when player responds? Returns message
	Chance      float64             // Base probability per check (0-1)
}

// getEventDefinitions returns all possible events with their properties
func getEventDefinitions() []eventDefinition {
	return []eventDefinition{
		{
			Type:     EventChasing,
			Emoji:    "🦋",
			Message:  "chasing a butterfly!",
			Duration: 10 * time.Minute,
			Condition: func(p *Pet) bool {
				return !p.Sleeping && p.Energy > 30 && (p.Mood == "playful" || p.Mood == "normal")
			},
			OnIgnored: func(p *Pet) {
				// No penalty, butterfly flew away
			},
			OnResponded: func(p *Pet) string {
				p.Happiness = min(p.Happiness+10, maxStat)
				p.Energy = max(p.Energy-5, minStat)
				return "🎉 You watched together! (+10 happiness)"
			},
			Chance: 0.15,
		},
		{
			Type:     EventFoundSomething,
			Emoji:    "🎁",
			Message:  "found something interesting!",
			Duration: 15 * time.Minute,
			Condition: func(p *Pet) bool {
				return !p.Sleeping && p.Energy > 20
			},
			OnIgnored: func(p *Pet) {
				// 50% chance it was something bad
				if randFloat64() < 0.5 {
					p.Health = max(p.Health-10, minStat)
				}
			},
			OnResponded: func(p *Pet) string {
				roll := randFloat64()
				if roll < 0.5 {
					p.Happiness = min(p.Happiness+15, maxStat)
					return "🧸 It was a fun toy! (+15 happiness)"
				} else if roll < 0.8 {
					p.Hunger = min(p.Hunger+20, maxStat)
					return "🍪 It was a tasty treat! (+20 hunger)"
				} else {
					p.Health = max(p.Health-5, minStat)
					return "🗑️ It was trash... you threw it away. (-5 health)"
				}
			},
			Chance: 0.1,
		},
		{
			Type:     EventScared,
			Emoji:    "⚡",
			Message:  "is scared of loud noises!",
			Duration: 5 * time.Minute,
			Condition: func(p *Pet) bool {
				return !p.Sleeping && p.Happiness < 70
			},
			OnIgnored: func(p *Pet) {
				p.Happiness = max(p.Happiness-15, minStat)
			},
			OnResponded: func(p *Pet) string {
				p.Happiness = min(p.Happiness+20, maxStat)
				return "🤗 You comforted them! (+20 happiness)"
			},
			Chance: 0.08,
		},
		{
			Type:     EventDaydreaming,
			Emoji:    "💭",
			Message:  "is daydreaming...",
			Duration: 8 * time.Minute,
			Condition: func(p *Pet) bool {
				return !p.Sleeping && p.Happiness > 50 && p.Energy > 40
			},
			OnIgnored: func(p *Pet) {
				// No penalty
			},
			OnResponded: func(p *Pet) string {
				thoughts := []string{
					"💭 Dreaming about endless treats...",
					"💭 Imagining a world of soft pillows...",
					"💭 Thinking about that butterfly...",
					"💭 Wondering what's beyond the window...",
					"💭 Planning world domination (cutely)...",
				}
				return thoughts[int(randFloat64()*float64(len(thoughts)))]
			},
			Chance: 0.12,
		},
		{
			Type:     EventAteSomething,
			Emoji:    "🤢",
			Message:  "ate something weird!",
			Duration: 10 * time.Minute,
			Condition: func(p *Pet) bool {
				return !p.Sleeping && p.Hunger < 50
			},
			OnIgnored: func(p *Pet) {
				p.Health = max(p.Health-20, minStat)
				p.Illness = true
			},
			OnResponded: func(p *Pet) string {
				p.Health = max(p.Health-5, minStat)
				return "💊 You gave them medicine just in time! (-5 health only)"
			},
			Chance: 0.05,
		},
		{
			Type:     EventSinging,
			Emoji:    "🎵",
			Message:  "is singing happily!",
			Duration: 5 * time.Minute,
			Condition: func(p *Pet) bool {
				return !p.Sleeping && p.Happiness > 80 && p.Energy > 60
			},
			OnIgnored: func(p *Pet) {
				// No penalty, rare happy moment
			},
			OnResponded: func(p *Pet) string {
				p.Happiness = min(p.Happiness+5, maxStat)
				return "🎶 You sang along! What a moment! (+5 happiness)"
			},
			Chance: 0.03, // Rare
		},
		{
			Type:     EventNightmare,
			Emoji:    "😰",
			Message:  "is having a nightmare!",
			Duration: 5 * time.Minute,
			Condition: func(p *Pet) bool {
				return p.Sleeping && p.Happiness < 60
			},
			OnIgnored: func(p *Pet) {
				p.Happiness = max(p.Happiness-20, minStat)
				p.Energy = max(p.Energy-10, minStat)
			},
			OnResponded: func(p *Pet) string {
				p.Sleeping = false
				p.AutoSleepTime = nil
				p.Happiness = min(p.Happiness+10, maxStat)
				return "🌙 You woke them gently. They feel safe now. (+10 happiness)"
			},
			Chance: 0.1,
		},
		{
			Type:     EventZoomies,
			Emoji:    "💨",
			Message:  "has the zoomies!",
			Duration: 3 * time.Minute,
			Condition: func(p *Pet) bool {
				return !p.Sleeping && p.Energy > 70 && p.Mood == "playful"
			},
			OnIgnored: func(p *Pet) {
				p.Energy = max(p.Energy-15, minStat)
				p.Happiness = min(p.Happiness+5, maxStat)
			},
			OnResponded: func(p *Pet) string {
				p.Energy = max(p.Energy-20, minStat)
				p.Happiness = min(p.Happiness+15, maxStat)
				return "🏃 You joined in! Exhausting but fun! (+15 happiness, -20 energy)"
			},
			Chance: 0.1,
		},
		{
			Type:     EventCuddles,
			Emoji:    "🥺",
			Message:  "wants cuddles!",
			Duration: 10 * time.Minute,
			Condition: func(p *Pet) bool {
				return !p.Sleeping && p.Mood == "needy"
			},
			OnIgnored: func(p *Pet) {
				p.Happiness = max(p.Happiness-10, minStat)
			},
			OnResponded: func(p *Pet) string {
				p.Happiness = min(p.Happiness+25, maxStat)
				p.Energy = min(p.Energy+5, maxStat)
				return "💕 Cuddle time! So cozy! (+25 happiness, +5 energy)"
			},
			Chance: 0.12,
		},
	}
}

// getEventDefinition returns the definition for a given event type
func getEventDefinition(eventType EventType) *eventDefinition {
	for _, def := range getEventDefinitions() {
		if def.Type == eventType {
			return &def
		}
	}
	return nil
}

// triggerRandomEvent attempts to trigger a random event based on conditions
func triggerRandomEvent(p *Pet) {
	now := timeNow()

	// Don't trigger if there's already an active event
	if p.CurrentEvent != nil && now.Before(p.CurrentEvent.ExpiresAt) {
		return
	}

	// If there was an expired event that wasn't responded to, apply consequences
	if p.CurrentEvent != nil && !p.CurrentEvent.Responded {
		def := getEventDefinition(p.CurrentEvent.Type)
		if def != nil && def.OnIgnored != nil {
			def.OnIgnored(p)
			log.Printf("Event %s was ignored, applying consequences", p.CurrentEvent.Type)
		}
		// Log the ignored event
		p.EventLog = append(p.EventLog, EventLogEntry{
			Type:       p.CurrentEvent.Type,
			Time:       p.CurrentEvent.StartTime,
			WasIgnored: true,
		})
		p.CurrentEvent = nil
	}

	// Dead pets don't get events
	if p.Dead {
		return
	}

	// Try to trigger a new event
	definitions := getEventDefinitions()
	for _, def := range definitions {
		if def.Condition(p) && randFloat64() < def.Chance {
			p.CurrentEvent = &Event{
				Type:      def.Type,
				StartTime: now,
				ExpiresAt: now.Add(def.Duration),
				Responded: false,
			}
			log.Printf("Event triggered: %s %s", def.Emoji, def.Message)
			return // Only one event at a time
		}
	}
}

// respondToEvent handles the player responding to the current event
func (p *Pet) respondToEvent() string {
	if p.CurrentEvent == nil || p.CurrentEvent.Responded {
		return ""
	}

	def := getEventDefinition(p.CurrentEvent.Type)
	if def == nil {
		return ""
	}

	var message string
	if def.OnResponded != nil {
		message = def.OnResponded(p)
	}

	p.CurrentEvent.Responded = true

	// Log the responded event
	p.EventLog = append(p.EventLog, EventLogEntry{
		Type:       p.CurrentEvent.Type,
		Time:       p.CurrentEvent.StartTime,
		WasIgnored: false,
	})

	// Keep only last 20 events in log
	if len(p.EventLog) > 20 {
		p.EventLog = p.EventLog[len(p.EventLog)-20:]
	}

	return message
}

// getEventDisplay returns the display string for the current event
func (p *Pet) getEventDisplay() (emoji, message string, hasEvent bool) {
	if p.CurrentEvent == nil || timeNow().After(p.CurrentEvent.ExpiresAt) {
		return "", "", false
	}

	def := getEventDefinition(p.CurrentEvent.Type)
	if def == nil {
		return "", "", false
	}

	if p.CurrentEvent.Responded {
		return def.Emoji, "event completed", false
	}

	return def.Emoji, def.Message, true
}

func (m *model) updateHourlyStats(t time.Time) {
	m.modifyStats(func(p *Pet) {
		// Record stat checkpoint every hour for evolution tracking
		if int(t.Minute()) == 0 {
			p.recordStatCheckpoint()
		}

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
			Form:      FormBaby,
			Sleeping:  false,
			LastSaved: now,
			Illness:   false,
		}
	}

	// Initialize evolution tracking maps
	if pet.Form == 0 {
		pet.Form = FormBaby // Ensure form is set for existing pets
	}
	if pet.CareQualityHistory == nil {
		pet.CareQualityHistory = make(map[int]CareQuality)
	}
	if pet.StatCheckpoints == nil {
		pet.StatCheckpoints = make(map[string][]StatCheck)
	}

	// Assign random chronotype at birth
	if pet.Chronotype == "" {
		pet.Chronotype = assignRandomChronotype()
		log.Printf("Assigned chronotype: %s", getChronotypeName(pet.Chronotype))
	}

	// Assign random personality traits at birth
	if len(pet.Traits) == 0 {
		pet.Traits = generateTraits()
	}

	// Initialize bond for new pets
	if pet.Bond == 0 {
		pet.Bond = initialBond
		log.Printf("Initialized bond at %d", initialBond)
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
	elapsedHours := elapsed.Hours()

	// Store current status before updates
	oldStatus := pet.LastStatus
	if oldStatus == "" {
		oldStatus = getStatus(pet)
	}

	// Update age and life stage
	// Calculate age from birth time to avoid drift from integer truncation
	birthTime := pet.Logs[0].Time
	pet.Age = int(now.Sub(birthTime).Hours())

	// Calculate life stage based on age and handle evolution
	oldLifeStage := pet.LifeStage
	if pet.Age < ageStageThresholds {
		pet.LifeStage = 0 // Baby
	} else if pet.Age < 2*ageStageThresholds {
		pet.LifeStage = 1 // Child
	} else {
		pet.LifeStage = 2 // Adult
	}

	// Handle evolution when life stage changes
	if oldLifeStage != pet.LifeStage && pet.LifeStage > 0 {
		pet.evolve(pet.LifeStage)
	}

	// Check death condition first
	if pet.Dead {
		return pet
	}

	// Calculate hunger decrease with trait modifiers
	hungerRate := float64(hungerDecreaseRate)
	if pet.Sleeping {
		hungerRate = float64(sleepingHungerRate)
	}
	hungerRate *= pet.getTraitModifier("hunger_decay")
	hungerLoss := int(elapsedHours * hungerRate)
	pet.Hunger = max(pet.Hunger-hungerLoss, minStat)

	// Apply chronotype-based multipliers (use local time for user's actual hour)
	// In tests, timeNow() is mocked, so we use now.Local().Hour()
	currentHour := now.Local().Hour()
	isActive := isActiveHours(&pet, currentHour)

	if !pet.Sleeping {
		// Energy decreases when awake (every 2 hours)
		energyMult := 1.0
		if !isActive {
			// 50% faster energy drain outside active hours
			energyMult = outsideActiveEnergyMult
		}
		// Apply trait modifier
		energyMult *= pet.getTraitModifier("energy_decay")
		energyLoss := int((elapsedHours / 2.0) * float64(energyDecreaseRate) * energyMult)
		pet.Energy = max(pet.Energy-energyLoss, minStat)
	} else {
		// Energy recovers while sleeping
		recoveryMult := 1.0
		if !isActive {
			// 20% better recovery during preferred sleep hours
			recoveryMult = preferredSleepRecoveryMult
		}
		// Use fractional accumulator to handle frequent small updates
		exactGain := elapsedHours * float64(energyRecoveryRate) * recoveryMult
		pet.FractionalEnergy += exactGain
		// Only apply whole numbers to Energy
		wholeGain := int(pet.FractionalEnergy)
		pet.FractionalEnergy -= float64(wholeGain)
		pet.Energy = min(pet.Energy+wholeGain, maxStat)
	}

	// Update happiness if stats are low
	if pet.Hunger < lowStatThreshold || pet.Energy < lowStatThreshold {
		happinessRate := float64(happinessDecreaseRate) * pet.getTraitModifier("happiness_decay")
		happinessLoss := int(elapsedHours * happinessRate)
		pet.Happiness = max(pet.Happiness-happinessLoss, minStat)
	}

	// Update bond from neglect or interactions
	if len(pet.LastInteractions) > 0 {
		// Find most recent interaction
		var mostRecent time.Time
		for _, interaction := range pet.LastInteractions {
			if interaction.Time.After(mostRecent) {
				mostRecent = interaction.Time
			}
		}
		hoursSinceInteraction := now.Sub(mostRecent).Hours()

		// Bond decays after bondDecayThreshold hours without interaction
		if hoursSinceInteraction > bondDecayThreshold {
			excessHours := hoursSinceInteraction - bondDecayThreshold
			bondLoss := int(excessHours / 12) * bondDecayRate // -1 per 12 hours of neglect
			if bondLoss > 0 {
				pet.Bond = max(pet.Bond-bondLoss, 0)
				log.Printf("Bond decreased by %d from neglect (%.1f hours since last interaction)", bondLoss, hoursSinceInteraction)
			}
		}
	}

	// Check for random illness when health is low (with trait and bond modifiers)
	if pet.Health < 50 && !pet.Illness {
		adjustedIllnessChance := illnessChance * pet.getTraitModifier("illness_chance")
		// High bond reduces illness chance
		if pet.Bond >= illnessResistanceBond {
			bondReduction := 1.0 - (float64(pet.Bond-illnessResistanceBond) / float64(maxBond-illnessResistanceBond) * 0.5)
			adjustedIllnessChance *= bondReduction
		}
		if randFloat64() < adjustedIllnessChance {
			pet.Illness = true
		}
	} else if pet.Health >= 50 {
		// Clear illness when health returns to safe levels
		pet.Illness = false
	}

	// Health decreases when any stat is critically low (with trait modifier)
	if pet.Hunger < 15 || pet.Happiness < 15 || pet.Energy < 15 {
		healthRate := 2.0 // 2%/hr when awake
		if pet.Sleeping {
			healthRate = 1.0 // 1%/hr when sleeping
		}
		healthRate *= pet.getTraitModifier("health_decay")
		healthLoss := int(elapsedHours * healthRate)
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

	// Apply autonomous behavior (auto-sleep, auto-wake, mood changes)
	if !pet.Dead {
		applyAutonomousBehavior(&pet)
	}

	// Trigger random life events
	triggerRandomEvent(&pet)

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
		// Handle cheat menu input
		if m.inCheatMenu {
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "c", "esc":
				m.inCheatMenu = false
				return m, nil
			case "up", "k":
				if m.cheatChoice > 0 {
					m.cheatChoice--
				}
			case "down", "j":
				if m.cheatChoice < len(cheatMenuOptions)-1 {
					m.cheatChoice++
				}
			case "enter", " ":
				m.executeCheat()
				if m.cheatChoice != 15 { // Not "Back"
					return m, nil
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "c":
			// Toggle cheat menu (hidden)
			if !m.pet.Dead {
				m.inCheatMenu = true
				m.cheatChoice = 0
				return m, nil
			}
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
		case "e", "r":
			// Respond to current event
			if m.pet.CurrentEvent != nil && !m.pet.CurrentEvent.Responded {
				result := m.pet.respondToEvent()
				if result != "" {
					m.setMessage(result)
				}
				saveState(&m.pet)
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
	if m.inCheatMenu {
		return m.renderCheatMenu()
	}

	// Build the view from components
	formEmoji := m.pet.getFormEmoji()
	title := gameStyles.title.Render(formEmoji + " " + m.pet.Name + " " + formEmoji)
	stats := m.renderStats()
	status := m.renderStatus()
	menu := m.renderMenu()

	// Show message if not expired
	var messageView string
	if m.message != "" && timeNow().Before(m.messageExpires) {
		messageView = gameStyles.status.Render(m.message)
	}

	// Check for active event
	var eventView string
	emoji, eventMsg, hasEvent := m.pet.getEventDisplay()
	if hasEvent {
		eventView = gameStyles.title.Render(fmt.Sprintf("✨ %s %s %s ✨", emoji, m.pet.Name+" is "+eventMsg, emoji))
	}

	// Join all sections vertically
	sections := []string{
		title,
		"",
		stats,
		"",
		status,
	}

	// Add event notification if present (before message)
	if eventView != "" {
		sections = append(sections, "", eventView, gameStyles.status.Render("Press [E] to respond!"))
	}

	// Add message if present
	if messageView != "" {
		sections = append(sections, "", messageView)
	}

	// Build help text
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

func (m model) renderStats() string {
	// Capitalize mood for display
	mood := m.pet.Mood
	if mood == "" {
		mood = "normal"
	}
	moodDisplay := strings.ToUpper(mood[:1]) + mood[1:]

	// Chronotype display with active hours
	chronoEmoji := getChronotypeEmoji(m.pet.Chronotype)
	chronoName := getChronotypeName(m.pet.Chronotype)
	wakeHour, sleepHour := getChronotypeSchedule(m.pet.Chronotype)
	chronoDisplay := fmt.Sprintf("%s %s (%d:00-%d:00)", chronoEmoji, chronoName, wakeHour, sleepHour)

	// Build trait display (just names, comma-separated)
	var traitNames []string
	for _, trait := range m.pet.Traits {
		traitNames = append(traitNames, trait.Name)
	}
	traitDisplay := strings.Join(traitNames, ", ")
	if traitDisplay == "" {
		traitDisplay = "None"
	}

	stats := []struct {
		name, value string
	}{
		{"Form", m.pet.getFormName()},
		{"Type", chronoDisplay},
		{"Traits", traitDisplay},
		{"Bond", getBondDescription(m.pet.Bond)},
		{"Mood", moodDisplay},
		{"Hunger", fmt.Sprintf("%d%%", m.pet.Hunger)},
		{"Happiness", fmt.Sprintf("%d%%", m.pet.Happiness)},
		{"Energy", fmt.Sprintf("%d%%", m.pet.Energy)},
		{"Health", fmt.Sprintf("%d%%", m.pet.Health)},
		{"Age", fmt.Sprintf("%dh", m.pet.Age)},
		{"Illness", map[bool]string{true: "Yes", false: "No"}[m.pet.Illness]},
	}

	var lines []string
	for _, stat := range stats {
		lines = append(lines, fmt.Sprintf("%-10s %s", stat.name+":", stat.value))
	}

	return gameStyles.stats.Render(strings.Join(lines, "\n"))
}

func (m model) renderStatus() string {
	return gameStyles.status.Render(fmt.Sprintf("Status: %s", getStatusWithLabel(m.pet)))
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

// Cheat menu options
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

func (m model) renderCheatMenu() string {
	var menuItems []string
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF0000")).
		Render("⚠️  CHEAT MENU ⚠️")

	for i, choice := range cheatMenuOptions {
		cursor := " "
		if m.cheatChoice == i {
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

func (m *model) executeCheat() {
	switch m.cheatChoice {
	case 0: // Max All Stats
		m.modifyStats(func(p *Pet) {
			p.Hunger = maxStat
			p.Happiness = maxStat
			p.Energy = maxStat
			p.Health = maxStat
		})
		m.setMessage("🎮 All stats maxed!")
	case 1: // Min All Stats (Critical)
		m.modifyStats(func(p *Pet) {
			p.Hunger = 10
			p.Happiness = 10
			p.Energy = 10
			p.Health = 10
		})
		m.setMessage("🎮 Stats set to critical!")
	case 2: // Full Energy
		m.modifyStats(func(p *Pet) {
			p.Energy = maxStat
		})
		m.setMessage("🎮 Energy maxed!")
	case 3: // Empty Energy (Auto-sleep)
		m.modifyStats(func(p *Pet) {
			p.Energy = 0
		})
		m.setMessage("🎮 Energy emptied!")
	case 4: // Mood: Normal
		m.modifyStats(func(p *Pet) {
			p.Mood = "normal"
			p.MoodExpiresAt = nil
		})
		m.setMessage("🎮 Mood set to normal")
	case 5: // Mood: Playful
		m.modifyStats(func(p *Pet) {
			p.Mood = "playful"
			p.MoodExpiresAt = nil
		})
		m.setMessage("🎮 Mood set to playful")
	case 6: // Mood: Lazy
		m.modifyStats(func(p *Pet) {
			p.Mood = "lazy"
			p.MoodExpiresAt = nil
		})
		m.setMessage("🎮 Mood set to lazy")
	case 7: // Mood: Needy
		m.modifyStats(func(p *Pet) {
			p.Mood = "needy"
			p.MoodExpiresAt = nil
		})
		m.setMessage("🎮 Mood set to needy")
	case 8: // Toggle Illness
		m.modifyStats(func(p *Pet) {
			p.Illness = !p.Illness
		})
		if m.pet.Illness {
			m.setMessage("🎮 Illness: ON")
		} else {
			m.setMessage("🎮 Illness: OFF")
		}
	case 9: // Toggle Sleep
		m.modifyStats(func(p *Pet) {
			p.Sleeping = !p.Sleeping
			p.AutoSleepTime = nil
			p.FractionalEnergy = 0 // Reset fractional accumulator
		})
		if m.pet.Sleeping {
			m.setMessage("🎮 Pet is now sleeping")
		} else {
			m.setMessage("🎮 Pet is now awake")
		}
	case 10: // Type: Early Bird
		m.modifyStats(func(p *Pet) {
			p.Chronotype = ChronotypeEarlyBird
		})
		m.setMessage("🎮 Type: 🌅 Early Bird (5am-9pm)")
	case 11: // Type: Normal
		m.modifyStats(func(p *Pet) {
			p.Chronotype = ChronotypeNormal
		})
		m.setMessage("🎮 Type: ☀️ Normal (7am-11pm)")
	case 12: // Type: Night Owl
		m.modifyStats(func(p *Pet) {
			p.Chronotype = ChronotypeNightOwl
		})
		m.setMessage("🎮 Type: 🦉 Night Owl (10am-2am)")
	case 13: // Age +24h
		m.modifyStats(func(p *Pet) {
			// Shift birth time back by 24 hours
			if len(p.Logs) > 0 {
				p.Logs[0].Time = p.Logs[0].Time.Add(-24 * time.Hour)
			}
		})
		m.setMessage(fmt.Sprintf("🎮 Age advanced! Now %dh", m.pet.Age))
	case 14: // Kill Pet
		m.modifyStats(func(p *Pet) {
			p.Dead = true
			p.CauseOfDeath = "Cheats"
		})
		m.setMessage("🎮 Pet has been killed")
		m.showingAdoptPrompt = true
	case 15: // Back
		m.inCheatMenu = false
	}
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
		return "💀"
	}

	// Icon 1: Activity (what pet is DOING)
	var activity string

	// Check for active event first
	if p.CurrentEvent != nil && !p.CurrentEvent.Responded && timeNow().Before(p.CurrentEvent.ExpiresAt) {
		def := getEventDefinition(p.CurrentEvent.Type)
		if def != nil {
			activity = def.Emoji
		}
	}

	// If no event, show sleep or awake state
	if activity == "" {
		if p.Sleeping {
			activity = "😴"
		} else {
			activity = "😸"
		}
	}

	// Icon 2: Feeling (most critical need) - only show if there's an issue
	var feeling string

	// Find the lowest stat to prioritize critical issues
	lowestStat := p.Health
	lowestFeeling := "🤢" // Sick

	if p.Energy < lowestStat {
		lowestStat = p.Energy
		lowestFeeling = "😾" // Tired
	}
	if p.Hunger < lowestStat {
		lowestStat = p.Hunger
		lowestFeeling = "🙀" // Hungry
	}
	if p.Happiness < lowestStat {
		lowestStat = p.Happiness
		lowestFeeling = "😿" // Sad
	}

	// Show critical feeling if any stat < 30
	if lowestStat < 30 {
		feeling = lowestFeeling
	} else if p.Energy < drowsyThreshold && !p.Sleeping {
		// Show drowsy if not critical but energy getting low (and not sleeping)
		feeling = "🥱"
	}
	// Otherwise feeling stays empty (all is well)

	return activity + feeling
}

// getStatusWithLabel returns status with text labels for the UI
func getStatusWithLabel(p Pet) string {
	if p.Dead {
		return "💀 Dead"
	}

	status := getStatus(p)

	// Add descriptive label based on the icons
	switch {
	case strings.Contains(status, "😴") && len(status) > 4:
		return status + " Sleeping (needs care)"
	case strings.Contains(status, "😴"):
		return status + " Sleeping"
	case strings.Contains(status, "🦋"):
		return status + " Chasing!"
	case strings.Contains(status, "🎁"):
		return status + " Found something!"
	case strings.Contains(status, "⚡"):
		return status + " Scared!"
	case strings.Contains(status, "💭"):
		return status + " Daydreaming"
	case strings.Contains(status, "🤢") && strings.HasPrefix(status, "🤢"):
		return status + " Ate something!"
	case strings.Contains(status, "🎵"):
		return status + " Singing!"
	case strings.Contains(status, "😰"):
		return status + " Nightmare!"
	case strings.Contains(status, "💨"):
		return status + " Zoomies!"
	case strings.Contains(status, "🥺") && strings.HasPrefix(status, "🥺"):
		return status + " Wants cuddles!"
	case strings.Contains(status, "🙀"):
		return status + " Hungry"
	case strings.Contains(status, "😾"):
		return status + " Tired"
	case strings.Contains(status, "😿"):
		return status + " Sad"
	case strings.Contains(status, "🤢"):
		return status + " Sick"
	case strings.Contains(status, "🥱"):
		return status + " Drowsy"
	default:
		return status + " Happy"
	}
}

// statsModel is a simple Bubble Tea model for displaying stats
type statsModel struct {
	pet Pet
}

func (m statsModel) Init() tea.Cmd {
	return nil
}

func (m statsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Exit on any key press, including ESC
		return m, tea.Quit
	case tea.MouseMsg:
		// Exit on any mouse button press (not just click, but press down)
		if msg.Action == tea.MouseActionPress {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m statsModel) View() string {
	// Helper function to create progress bar
	makeBar := func(value int) string {
		filled := value / 20 // 5 blocks for 100%
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

	formEmoji := m.pet.getFormEmoji()
	formName := m.pet.getFormName()
	status := getStatus(m.pet)
	illnessStatus := "No"
	if m.pet.Illness {
		illnessStatus = "Yes"
	}

	// Chronotype display
	chronoEmoji := getChronotypeEmoji(m.pet.Chronotype)
	chronoName := getChronotypeName(m.pet.Chronotype)
	wakeHour, sleepHour := getChronotypeSchedule(m.pet.Chronotype)
	chronoDisplay := fmt.Sprintf("%s %s (%d:00-%d:00)", chronoEmoji, chronoName, wakeHour, sleepHour)

	// Traits display
	var traitNames []string
	for _, trait := range m.pet.Traits {
		traitNames = append(traitNames, trait.Name)
	}
	traitDisplay := strings.Join(traitNames, ", ")
	if traitDisplay == "" {
		traitDisplay = "None"
	}

	bondDisplay := getBondDescription(m.pet.Bond)

	var s strings.Builder
	s.WriteString("╔════════════════════════════════════╗\n")
	s.WriteString(fmt.Sprintf("║  %s %s %s                  ║\n", formEmoji, m.pet.Name, formEmoji))
	s.WriteString("╠════════════════════════════════════╣\n")
	s.WriteString(fmt.Sprintf("║  Form:    %-24s ║\n", formName))
	s.WriteString(fmt.Sprintf("║  Type:    %-24s ║\n", chronoDisplay))
	s.WriteString(fmt.Sprintf("║  Traits:  %-24s ║\n", traitDisplay))
	s.WriteString(fmt.Sprintf("║  Bond:    %-24s ║\n", bondDisplay))
	s.WriteString(fmt.Sprintf("║  Age:     %-24s ║\n", fmt.Sprintf("%d hours", m.pet.Age)))
	s.WriteString(fmt.Sprintf("║  Status:  %-24s ║\n", status))
	s.WriteString("║                                    ║\n")
	s.WriteString(fmt.Sprintf("║  Hunger:    [%s] %3d%%           ║\n", makeBar(m.pet.Hunger), m.pet.Hunger))
	s.WriteString(fmt.Sprintf("║  Happiness: [%s] %3d%%           ║\n", makeBar(m.pet.Happiness), m.pet.Happiness))
	s.WriteString(fmt.Sprintf("║  Energy:    [%s] %3d%%           ║\n", makeBar(m.pet.Energy), m.pet.Energy))
	s.WriteString(fmt.Sprintf("║  Health:    [%s] %3d%%           ║\n", makeBar(m.pet.Health), m.pet.Health))
	s.WriteString("║                                    ║\n")
	s.WriteString(fmt.Sprintf("║  Illness:   %-23s║\n", illnessStatus))
	s.WriteString("╚════════════════════════════════════╝\n")
	s.WriteString("\nPress ESC, click, or any key to close...")

	return s.String()
}

func displayStats(pet Pet) {
	p := tea.NewProgram(statsModel{pet: pet}, tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running stats display: %v\n", err)
		os.Exit(1)
	}
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
	// Configure logging to write to config directory
	configDir := filepath.Dir(getConfigPath())
	logFile := filepath.Join(configDir, "vpet.log")
	logFileHandle, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer logFileHandle.Close()
	log.SetOutput(logFileHandle)

	updateOnly := flag.Bool("u", false, "Update pet stats only, don't run UI")
	statusFlag := flag.Bool("status", false, "Output current status emoji")
	statsFlag := flag.Bool("stats", false, "Display detailed pet statistics")
	flag.Parse()

	if *statsFlag {
		pet := loadState()
		displayStats(pet)
		return
	}

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
