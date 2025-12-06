package pet

import "strings"

// GetStatus returns the status emoji(s) for the pet
func GetStatus(p Pet) string {
	if p.Dead {
		return "💀"
	}

	// Icon 1: Activity (what pet is DOING)
	var activity string

	// Check for active event first
	if p.CurrentEvent != nil && !p.CurrentEvent.Responded && TimeNow().Before(p.CurrentEvent.ExpiresAt) {
		def := GetEventDefinition(p.CurrentEvent.Type)
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

	// Icon 2: Feeling (most critical need)
	var feeling string

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
	} else if p.Energy < DrowsyThreshold && !p.Sleeping {
		feeling = "🥱"
	}

	// If no critical feeling, show the most pressing want
	if feeling == "" {
		if want := GetWantEmoji(p); want != "" {
			return activity + want
		}
	}

	return activity + feeling
}

// GetStatusWithLabel returns status with text labels for the UI
func GetStatusWithLabel(p Pet) string {
	if p.Dead {
		return "💀 Dead"
	}

	status := GetStatus(p)

	switch {
	case strings.Contains(status, "😴") && strings.Contains(status, "😾"):
		return status + " Sleeping"
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
