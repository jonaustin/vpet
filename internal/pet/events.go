package pet

import (
	"log"
	"time"
)

// Event type constants
const (
	EventNone           = ""
	EventChasing        = "chasing"
	EventFoundSomething = "found"
	EventScared         = "scared"
	EventDaydreaming    = "daydreaming"
	EventAteSomething   = "ate_something"
	EventSinging        = "singing"
	EventNightmare      = "nightmare"
	EventLearnedTrick   = "learned_trick"
	EventZoomies        = "zoomies"
	EventCuddles        = "cuddles"
)

// EventDefinition describes an event's properties and conditions
type EventDefinition struct {
	Type        string
	Emoji       string
	Message     string
	Duration    time.Duration
	Condition   func(p *Pet) bool
	OnIgnored   func(p *Pet)
	OnResponded func(p *Pet) string
	Chance      float64
}

// GetEventDefinitions returns all possible events with their properties
func GetEventDefinitions() []EventDefinition {
	return []EventDefinition{
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
				p.Happiness = min(p.Happiness+10, MaxStat)
				p.Energy = max(p.Energy-5, MinStat)
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
				if RandFloat64() < 0.5 {
					p.Health = max(p.Health-10, MinStat)
				}
			},
			OnResponded: func(p *Pet) string {
				roll := RandFloat64()
				if roll < 0.5 {
					p.Happiness = min(p.Happiness+15, MaxStat)
					return "🧸 It was a fun toy! (+15 happiness)"
				} else if roll < 0.8 {
					p.Hunger = min(p.Hunger+20, MaxStat)
					return "🍪 It was a tasty treat! (+20 hunger)"
				} else {
					p.Health = max(p.Health-5, MinStat)
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
				p.Happiness = max(p.Happiness-15, MinStat)
			},
			OnResponded: func(p *Pet) string {
				p.Happiness = min(p.Happiness+20, MaxStat)
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
				return thoughts[int(RandFloat64()*float64(len(thoughts)))]
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
				p.Health = max(p.Health-20, MinStat)
				p.Illness = true
			},
			OnResponded: func(p *Pet) string {
				p.Health = max(p.Health-5, MinStat)
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
				p.Happiness = min(p.Happiness+5, MaxStat)
				return "🎶 You sang along! What a moment! (+5 happiness)"
			},
			Chance: 0.03,
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
				p.Happiness = max(p.Happiness-20, MinStat)
				p.Energy = max(p.Energy-10, MinStat)
			},
			OnResponded: func(p *Pet) string {
				p.Sleeping = false
				p.AutoSleepTime = nil
				p.Happiness = min(p.Happiness+10, MaxStat)
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
				p.Energy = max(p.Energy-15, MinStat)
				p.Happiness = min(p.Happiness+5, MaxStat)
			},
			OnResponded: func(p *Pet) string {
				p.Energy = max(p.Energy-20, MinStat)
				p.Happiness = min(p.Happiness+15, MaxStat)
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
				p.Happiness = max(p.Happiness-10, MinStat)
			},
			OnResponded: func(p *Pet) string {
				p.Happiness = min(p.Happiness+25, MaxStat)
				p.Energy = min(p.Energy+5, MaxStat)
				return "💕 Cuddle time! So cozy! (+25 happiness, +5 energy)"
			},
			Chance: 0.12,
		},
	}
}

// GetEventDefinition returns the definition for a given event type
func GetEventDefinition(eventType string) *EventDefinition {
	for _, def := range GetEventDefinitions() {
		if def.Type == eventType {
			return &def
		}
	}
	return nil
}

// TriggerRandomEvent attempts to trigger a random event based on conditions
func TriggerRandomEvent(p *Pet) {
	now := TimeNow()

	// Don't trigger if there's already an active event
	if p.CurrentEvent != nil && now.Before(p.CurrentEvent.ExpiresAt) {
		return
	}

	// If there was an expired event that wasn't responded to, apply consequences
	if p.CurrentEvent != nil && !p.CurrentEvent.Responded {
		def := GetEventDefinition(p.CurrentEvent.Type)
		if def != nil && def.OnIgnored != nil {
			def.OnIgnored(p)
			log.Printf("Event %s was ignored, applying consequences", p.CurrentEvent.Type)
		}
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
	definitions := GetEventDefinitions()
	for _, def := range definitions {
		if def.Condition(p) && RandFloat64() < def.Chance {
			p.CurrentEvent = &Event{
				Type:      def.Type,
				StartTime: now,
				ExpiresAt: now.Add(def.Duration),
				Responded: false,
			}
			log.Printf("Event triggered: %s %s", def.Emoji, def.Message)
			return
		}
	}
}

// RespondToEvent handles the player responding to the current event
func (p *Pet) RespondToEvent() string {
	if p.CurrentEvent == nil || p.CurrentEvent.Responded {
		return ""
	}

	def := GetEventDefinition(p.CurrentEvent.Type)
	if def == nil {
		return ""
	}

	var message string
	if def.OnResponded != nil {
		message = def.OnResponded(p)
	}

	p.CurrentEvent.Responded = true

	p.EventLog = append(p.EventLog, EventLogEntry{
		Type:       p.CurrentEvent.Type,
		Time:       p.CurrentEvent.StartTime,
		WasIgnored: false,
	})

	if len(p.EventLog) > 20 {
		p.EventLog = p.EventLog[len(p.EventLog)-20:]
	}

	return message
}

// GetEventDisplay returns the display string for the current event
func (p *Pet) GetEventDisplay() (emoji, message string, hasEvent bool) {
	if p.CurrentEvent == nil || TimeNow().After(p.CurrentEvent.ExpiresAt) {
		return "", "", false
	}

	def := GetEventDefinition(p.CurrentEvent.Type)
	if def == nil {
		return "", "", false
	}

	if p.CurrentEvent.Responded {
		return def.Emoji, "event completed", false
	}

	return def.Emoji, def.Message, true
}
