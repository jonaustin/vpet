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
		p.Health = min(p.Health+medicineEffect, maxStat)
		log.Printf("Administered medicine. Health is now %d", p.Health)
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

	m.modifyStats(func(p *Pet) {
		p.Sleeping = false
		// Clear auto-sleep time when woken by feeding
		p.AutoSleepTime = nil
		p.Hunger = min(p.Hunger+feedHungerIncrease, maxStat)
		p.Happiness = min(p.Happiness+feedHappinessIncrease, maxStat)
		log.Printf("Fed pet. Hunger is now %d, Happiness is now %d", p.Hunger, p.Happiness)
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

	m.modifyStats(func(p *Pet) {
		p.Sleeping = false
		// Clear auto-sleep time when woken by playing
		p.AutoSleepTime = nil
		p.Happiness = min(p.Happiness+playHappinessIncrease, maxStat)
		p.Energy = max(p.Energy-playEnergyDecrease, minStat)
		p.Hunger = max(p.Hunger-playHungerDecrease, minStat)
		log.Printf("Played with pet. Happiness is now %d, Energy is now %d, Hunger is now %d", p.Happiness, p.Energy, p.Hunger)
	})

	// Different messages based on mood
	if m.pet.Mood == "playful" {
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
		log.Printf("Pet is now sleeping: %t", p.Sleeping)
	})
}

// applyAutonomousBehavior makes the pet act on its own based on current state
func applyAutonomousBehavior(p *Pet) {
	now := timeNow()

	// Auto-sleep when exhausted
	if p.Energy <= autoSleepThreshold && !p.Sleeping && !p.Dead {
		p.Sleeping = true
		p.AutoSleepTime = &now
		log.Printf("Pet fell asleep automatically due to low energy (%d)", p.Energy)
	}

	// Auto-wake after sufficient sleep and energy restored
	if p.Sleeping && p.AutoSleepTime != nil {
		sleepDuration := now.Sub(*p.AutoSleepTime)
		sleepHours := sleepDuration.Hours()

		// Wake up if: slept minimum hours AND energy is restored, OR slept maximum hours
		if (sleepHours >= minSleepDuration && p.Energy >= autoWakeEnergy) || sleepHours >= maxSleepDuration {
			p.Sleeping = false
			p.AutoSleepTime = nil
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

	// Calculate hunger decrease
	hungerRate := hungerDecreaseRate
	if pet.Sleeping {
		hungerRate = sleepingHungerRate
	}
	hungerLoss := int(elapsedHours * float64(hungerRate))
	pet.Hunger = max(pet.Hunger-hungerLoss, minStat)

	if !pet.Sleeping {
		// Energy decreases when awake (every 2 hours)
		energyLoss := int((elapsedHours / 2.0) * float64(energyDecreaseRate))
		pet.Energy = max(pet.Energy-energyLoss, minStat)
	} else {
		// Energy recovers while sleeping
		energyGain := int(elapsedHours * float64(energyRecoveryRate))
		pet.Energy = min(pet.Energy+energyGain, maxStat)
	}

	// Update happiness if stats are low
	if pet.Hunger < lowStatThreshold || pet.Energy < lowStatThreshold {
		happinessLoss := int(elapsedHours * float64(happinessDecreaseRate))
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
		healthLoss := int(elapsedHours * float64(healthRate))
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

	stats := []struct {
		name, value string
	}{
		{"Form", m.pet.getFormName()},
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

	// Show drowsy warning when energy is getting low
	if p.Energy < drowsyThreshold {
		return "🥱 Drowsy"
	}

	// Show mood-based status when pet is generally happy
	switch p.Mood {
	case "playful":
		return "🎾 Playful"
	case "lazy":
		return "😪 Lazy"
	case "needy":
		// Show what the pet wants based on lowest non-critical stat
		if p.Hunger < p.Happiness && p.Hunger < p.Energy {
			return "🍖 Wants Food"
		} else if p.Happiness < p.Energy {
			return "🎮 Wants Play"
		}
		return "🥺 Wants Attention"
	default:
		return "😸 Happy"
	}
}

func displayStats(pet Pet) {
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

	formEmoji := pet.getFormEmoji()
	formName := pet.getFormName()
	status := getStatus(pet)
	illnessStatus := "No"
	if pet.Illness {
		illnessStatus = "Yes"
	}

	// Display formatted stats box
	fmt.Println("╔════════════════════════════════════╗")
	fmt.Printf("║  %s %s %s                  ║\n", formEmoji, pet.Name, formEmoji)
	fmt.Println("╠════════════════════════════════════╣")
	fmt.Printf("║  Form:    %-24s ║\n", formName)
	fmt.Printf("║  Age:     %-24s ║\n", fmt.Sprintf("%d hours", pet.Age))
	fmt.Printf("║  Status:  %-24s ║\n", status)
	fmt.Println("║                                    ║")
	fmt.Printf("║  Hunger:    [%s] %3d%%           ║\n", makeBar(pet.Hunger), pet.Hunger)
	fmt.Printf("║  Happiness: [%s] %3d%%           ║\n", makeBar(pet.Happiness), pet.Happiness)
	fmt.Printf("║  Energy:    [%s] %3d%%           ║\n", makeBar(pet.Energy), pet.Energy)
	fmt.Printf("║  Health:    [%s] %3d%%           ║\n", makeBar(pet.Health), pet.Health)
	fmt.Println("║                                    ║")
	fmt.Printf("║  Illness:   %-23s║\n", illnessStatus)
	fmt.Println("╚════════════════════════════════════╝")
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
