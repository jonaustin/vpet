package pet

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

// Testable time and random functions
var (
	TimeNow     = func() time.Time { return time.Now().UTC() }
	RandFloat64 = rand.Float64
)

// LogEntry represents a status change event
type LogEntry struct {
	Time      time.Time `json:"time"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
}

// Trait represents a personality characteristic that affects pet behavior
type Trait struct {
	Name      string             `json:"name"`
	Category  string             `json:"category"`  // "temperament", "appetite", "sociability", "constitution"
	Modifiers map[string]float64 `json:"modifiers"` // stat_name -> multiplier
}

// Interaction represents a player action with the pet
type Interaction struct {
	Type string    `json:"type"` // "feed", "play", "medicine"
	Time time.Time `json:"time"`
}

// CareQuality tracks average stats during a life stage
type CareQuality struct {
	AvgHunger    int `json:"avg_hunger"`
	AvgHappiness int `json:"avg_happiness"`
	AvgEnergy    int `json:"avg_energy"`
	AvgHealth    int `json:"avg_health"`
}

// StatCheck records stats at a point in time for averaging
type StatCheck struct {
	Time      time.Time `json:"time"`
	Hunger    int       `json:"hunger"`
	Happiness int       `json:"happiness"`
	Energy    int       `json:"energy"`
	Health    int       `json:"health"`
}

// Event represents a life event happening to the pet
type Event struct {
	Type      string    `json:"type"`
	StartTime time.Time `json:"start_time"`
	ExpiresAt time.Time `json:"expires_at"`
	Responded bool      `json:"responded"`
}

// EventLogEntry records past events for the pet's "memory"
type EventLogEntry struct {
	Type       string    `json:"type"`
	Time       time.Time `json:"time"`
	WasIgnored bool      `json:"was_ignored"`
}

// Pet represents the virtual pet's state
type Pet struct {
	Name               string                 `json:"name"`
	Hunger             int                    `json:"hunger"`
	Happiness          int                    `json:"happiness"`
	Energy             int                    `json:"energy"`
	Health             int                    `json:"health"`
	Age                int                    `json:"age"`
	LifeStage          int                    `json:"stage"`
	Form               PetForm                `json:"form"`
	Sleeping           bool                   `json:"sleeping"`
	Dead               bool                   `json:"dead"`
	CauseOfDeath       string                 `json:"cause_of_death,omitempty"`
	LastSaved          time.Time              `json:"last_saved"`
	CriticalStartTime  *time.Time             `json:"critical_start_time,omitempty"`
	Illness            bool                   `json:"illness"`
	LastStatus         string                 `json:"last_status,omitempty"`
	Logs               []LogEntry             `json:"logs,omitempty"`
	CareQualityHistory map[int]CareQuality    `json:"care_quality_history,omitempty"`
	StatCheckpoints    map[string][]StatCheck `json:"stat_checkpoints,omitempty"`

	// Autonomous behavior fields
	Mood          string     `json:"mood,omitempty"`
	MoodExpiresAt *time.Time `json:"mood_expires_at,omitempty"`
	AutoSleepTime *time.Time `json:"auto_sleep_time,omitempty"`

	// Life events system
	CurrentEvent *Event          `json:"current_event,omitempty"`
	EventLog     []EventLogEntry `json:"event_log,omitempty"`

	// Circadian rhythm
	Chronotype string `json:"chronotype,omitempty"`

	// Personality traits
	Traits []Trait `json:"traits,omitempty"`

	// Bonding system
	Bond             int           `json:"bond,omitempty"`
	LastInteractions []Interaction `json:"last_interactions,omitempty"`

	// Fractional stat accumulators
	FractionalEnergy float64 `json:"fractional_energy,omitempty"`
}

// RecordStatCheckpoint records current stats for evolution tracking
func (p *Pet) RecordStatCheckpoint() {
	if p.StatCheckpoints == nil {
		p.StatCheckpoints = make(map[string][]StatCheck)
	}

	stageKey := fmt.Sprintf("stage_%d", p.LifeStage)
	checkpoint := StatCheck{
		Time:      TimeNow(),
		Hunger:    p.Hunger,
		Happiness: p.Happiness,
		Energy:    p.Energy,
		Health:    p.Health,
	}

	p.StatCheckpoints[stageKey] = append(p.StatCheckpoints[stageKey], checkpoint)
}

// CalculateCareQuality calculates average care quality for a life stage
func (p *Pet) CalculateCareQuality(stage int) CareQuality {
	stageKey := fmt.Sprintf("stage_%d", stage)
	checkpoints := p.StatCheckpoints[stageKey]

	if len(checkpoints) == 0 {
		return CareQuality{
			AvgHunger:    MaxStat,
			AvgHappiness: MaxStat,
			AvgEnergy:    MaxStat,
			AvgHealth:    MaxStat,
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

// OverallAverage returns the average of all care quality stats
func (cq CareQuality) OverallAverage() int {
	return (cq.AvgHunger + cq.AvgHappiness + cq.AvgEnergy + cq.AvgHealth) / 4
}

// Evolve handles pet evolution when life stage changes
func (p *Pet) Evolve(newStage int) {
	prevStage := newStage - 1
	careQuality := p.CalculateCareQuality(prevStage)

	if p.CareQualityHistory == nil {
		p.CareQualityHistory = make(map[int]CareQuality)
	}
	p.CareQualityHistory[prevStage] = careQuality

	avgCare := careQuality.OverallAverage()

	switch newStage {
	case 1: // Evolving to Child
		if avgCare >= GoodCareThreshold {
			p.Form = FormHealthyChild
		} else if avgCare >= PoorCareThreshold {
			p.Form = FormTroubledChild
		} else {
			p.Form = FormSicklyChild
		}

	case 2: // Evolving to Adult
		switch p.Form {
		case FormHealthyChild:
			if avgCare >= PerfectCareThreshold {
				p.Form = FormEliteAdult
			} else if avgCare >= GoodCareThreshold {
				p.Form = FormStandardAdult
			} else {
				p.Form = FormGrumpyAdult
			}
		case FormTroubledChild:
			if avgCare >= GoodCareThreshold {
				p.Form = FormRedeemedAdult
			} else {
				p.Form = FormDelinquentAdult
			}
		case FormSicklyChild:
			p.Form = FormWeakAdult
		}
	}

	log.Printf("Pet evolved to %s (care quality: %d%%)", p.GetFormName(), avgCare)
}

// GetFormName returns the display name for the pet's current form
func (p *Pet) GetFormName() string {
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

// GetFormEmoji returns the emoji for the pet's current form
func (p *Pet) GetFormEmoji() string {
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

// GetTraitModifier returns the combined modifier for a given stat type
func (p *Pet) GetTraitModifier(modifierKey string) float64 {
	multiplier := 1.0
	for _, trait := range p.Traits {
		if mod, exists := trait.Modifiers[modifierKey]; exists {
			multiplier *= mod
		}
	}
	return multiplier
}

// GetBondMultiplier returns effectiveness multiplier based on bond level (0.5 to 1.0)
func (p *Pet) GetBondMultiplier() float64 {
	return MinBondMultiplier + (float64(p.Bond)/float64(MaxBond))*(MaxBondMultiplier-MinBondMultiplier)
}

// AddInteraction records an interaction and maintains history limit
func (p *Pet) AddInteraction(actionType string) {
	p.LastInteractions = append(p.LastInteractions, Interaction{
		Type: actionType,
		Time: TimeNow(),
	})
	if len(p.LastInteractions) > MaxInteractionHistory {
		p.LastInteractions = p.LastInteractions[len(p.LastInteractions)-MaxInteractionHistory:]
	}
}

// UpdateBond modifies bond level and clamps to valid range
func (p *Pet) UpdateBond(change int) {
	p.Bond = max(0, min(p.Bond+change, MaxBond))
	log.Printf("Bond changed by %d, now %d", change, p.Bond)
}

// CountRecentInteractions counts how many times an action occurred within a time window
func CountRecentInteractions(interactions []Interaction, actionType string, window time.Duration) int {
	now := TimeNow()
	count := 0
	for _, interaction := range interactions {
		if interaction.Type == actionType && now.Sub(interaction.Time) < window {
			count++
		}
	}
	return count
}

// GetBondDescription returns a descriptive label for the bond level
func GetBondDescription(bond int) string {
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

// Chronotype helpers

// GetChronotypeSchedule returns (wakeHour, sleepHour) for a chronotype
func GetChronotypeSchedule(chronotype string) (int, int) {
	switch chronotype {
	case ChronotypeEarlyBird:
		return 5, 21 // 5am - 9pm
	case ChronotypeNightOwl:
		return 10, 2 // 10am - 2am (next day)
	default: // ChronotypeNormal
		return 7, 23 // 7am - 11pm
	}
}

// IsActiveHours checks if the given hour is within the pet's active window
func IsActiveHours(p *Pet, hour int) bool {
	wakeHour, sleepHour := GetChronotypeSchedule(p.Chronotype)

	if sleepHour > wakeHour {
		return hour >= wakeHour && hour < sleepHour
	}
	return hour >= wakeHour || hour < sleepHour
}

// GetChronotypeName returns a display-friendly name
func GetChronotypeName(chronotype string) string {
	switch chronotype {
	case ChronotypeEarlyBird:
		return "Early Bird"
	case ChronotypeNightOwl:
		return "Night Owl"
	default:
		return "Normal"
	}
}

// GetChronotypeEmoji returns an emoji for the chronotype
func GetChronotypeEmoji(chronotype string) string {
	switch chronotype {
	case ChronotypeEarlyBird:
		return "🌅"
	case ChronotypeNightOwl:
		return "🦉"
	default:
		return "☀️"
	}
}

// AssignRandomChronotype picks a random chronotype for a new pet
func AssignRandomChronotype() string {
	roll := RandFloat64()
	if roll < 0.33 {
		return ChronotypeEarlyBird
	} else if roll < 0.66 {
		return ChronotypeNormal
	}
	return ChronotypeNightOwl
}

// GenerateTraits assigns random personality traits at birth
func GenerateTraits() []Trait {
	traitDefinitions := map[string][]Trait{
		"temperament": {
			{
				Name:     "Calm",
				Category: "temperament",
				Modifiers: map[string]float64{
					"energy_decay":    0.8,
					"happiness_decay": 0.85,
				},
			},
			{
				Name:     "Hyperactive",
				Category: "temperament",
				Modifiers: map[string]float64{
					"energy_decay": 1.3,
					"play_bonus":   1.25,
				},
			},
		},
		"appetite": {
			{
				Name:     "Picky",
				Category: "appetite",
				Modifiers: map[string]float64{
					"feed_bonus": 0.75,
				},
			},
			{
				Name:     "Hungry",
				Category: "appetite",
				Modifiers: map[string]float64{
					"hunger_decay": 1.2,
					"feed_bonus":   1.25,
				},
			},
		},
		"sociability": {
			{
				Name:     "Independent",
				Category: "sociability",
				Modifiers: map[string]float64{
					"happiness_decay": 0.75,
				},
			},
			{
				Name:     "Needy",
				Category: "sociability",
				Modifiers: map[string]float64{
					"happiness_decay":      1.15,
					"play_bonus":           1.2,
					"feed_bonus_happiness": 1.3,
				},
			},
		},
		"constitution": {
			{
				Name:     "Robust",
				Category: "constitution",
				Modifiers: map[string]float64{
					"illness_chance": 0.5,
					"health_decay":   0.85,
				},
			},
			{
				Name:     "Fragile",
				Category: "constitution",
				Modifiers: map[string]float64{
					"illness_chance": 1.8,
					"health_decay":   1.2,
				},
			},
		},
	}

	var traits []Trait
	for _, options := range traitDefinitions {
		index := int(RandFloat64() * float64(len(options)))
		if index >= len(options) {
			index = len(options) - 1
		}
		selectedTrait := options[index]
		traits = append(traits, selectedTrait)
		log.Printf("Assigned %s trait: %s", selectedTrait.Category, selectedTrait.Name)
	}

	return traits
}

// GetWantEmoji returns an icon for the pet's most pressing desire when idle
func GetWantEmoji(p Pet) string {
	if p.Dead || p.Sleeping {
		return ""
	}

	if p.CurrentEvent != nil && !p.CurrentEvent.Responded && TimeNow().Before(p.CurrentEvent.ExpiresAt) {
		return ""
	}

	type need struct {
		deficit   int
		emoji     string
		threshold int
	}

	needs := []need{
		{deficit: MaxStat - p.Hunger, emoji: "🍖", threshold: WantHungerThreshold},
		{deficit: MaxStat - p.Happiness, emoji: "🎾", threshold: WantHappyThreshold},
		{deficit: MaxStat - p.Energy, emoji: "🛌", threshold: WantEnergyThreshold},
	}

	best := need{deficit: 0}
	for _, n := range needs {
		if n.deficit >= n.threshold && n.deficit > best.deficit {
			best = n
		}
	}

	if best.deficit > 0 {
		return best.emoji
	}

	return ""
}

// Helper functions
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
