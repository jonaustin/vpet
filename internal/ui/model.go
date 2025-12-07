package ui

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"vpet/internal/pet"
)

// Model represents the game state
type Model struct {
	Pet                pet.Pet
	Choice             int
	Quitting           bool
	ShowingAdoptPrompt bool
	EvolutionMessage   string
	Message            string
	MessageExpires     time.Time
	InCheatMenu        bool
	CheatChoice        int
	Animation          Animation
}

type tickMsg time.Time
type animTickMsg struct {
	started time.Time
}

// NewModel creates a new game model
func NewModel() Model {
	p := pet.LoadState()
	return Model{
		Pet:                p,
		Choice:             0,
		ShowingAdoptPrompt: p.Dead,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(time.Minute, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func animTick(start time.Time) tea.Cmd {
	return tea.Tick(AnimationFrameDuration, func(t time.Time) tea.Msg {
		return animTickMsg{started: start}
	})
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// While an animation is playing, ignore inputs except quit keys
		if m.Animation.Type != AnimNone {
			switch msg.String() {
			case "ctrl+c", "q":
				m.Quitting = true
				return m, tea.Quit
			default:
				return m, nil
			}
		}

		// Handle cheat menu input
		if m.InCheatMenu {
			switch msg.String() {
			case "ctrl+c", "q":
				m.Quitting = true
				return m, tea.Quit
			case "c", "esc":
				m.InCheatMenu = false
				return m, nil
			case "up", "k":
				if m.CheatChoice > 0 {
					m.CheatChoice--
				}
			case "down", "j":
				if m.CheatChoice < len(cheatMenuOptions)-1 {
					m.CheatChoice++
				}
			case "enter", " ":
				m.executeCheat()
				if m.CheatChoice != 15 { // Not "Back"
					return m, nil
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		case "c":
			if !m.Pet.Dead {
				m.InCheatMenu = true
				m.CheatChoice = 0
				return m, nil
			}
		case "y":
			if m.Pet.Dead && m.ShowingAdoptPrompt {
				m.Pet = pet.NewPet(nil)
				m.ShowingAdoptPrompt = false
				m.Choice = 0
				pet.SaveState(&m.Pet)
				return m, nil
			}
		case "n":
			if m.Pet.Dead && m.ShowingAdoptPrompt {
				m.ShowingAdoptPrompt = false
				return m, nil
			}
		case "e", "r":
			if m.Pet.CurrentEvent != nil && !m.Pet.CurrentEvent.Responded {
				result := m.Pet.RespondToEvent()
				if result != "" {
					m.setMessage(result)
				}
				pet.SaveState(&m.Pet)
				return m, nil
			}
		case "up", "k":
			if m.Choice > 0 {
				m.Choice--
			}
		case "down", "j":
			if m.Choice < 4 {
				m.Choice++
			}
		case "enter", " ":
			if m.Pet.Dead {
				return m, nil
			}
			switch m.Choice {
			case 0:
				if m.feed() {
					return m, animTick(m.Animation.StartTime)
				}
			case 1:
				if m.play() {
					return m, animTick(m.Animation.StartTime)
				}
			case 2:
				if m.toggleSleep() {
					return m, animTick(m.Animation.StartTime)
				}
			case 3:
				if m.administerMedicine() {
					return m, animTick(m.Animation.StartTime)
				}
			case 4:
				m.Quitting = true
				return m, tea.Quit
			}
		}

	case tickMsg:
		m.updateHourlyStats(time.Time(msg))
		if m.Pet.Dead && !m.ShowingAdoptPrompt {
			m.ShowingAdoptPrompt = true
		}
		return m, tick()

	case animTickMsg:
		// Drop ticks that belong to an older animation (e.g., if a new action started)
		if m.Animation.Type == AnimNone || !m.Animation.StartTime.Equal(msg.started) {
			return m, nil
		}

		m.Animation.Frame++
		if IsAnimationComplete(m.Animation) {
			m.Animation = Animation{}
			return m, nil
		}

		return m, animTick(m.Animation.StartTime)
	}

	return m, nil
}

// Helper to modify stats and save
func (m *Model) modifyStats(f func(*pet.Pet)) {
	f(&m.Pet)
	pet.SaveState(&m.Pet)
}

func (m *Model) setMessage(msg string) {
	m.Message = msg
	m.MessageExpires = pet.TimeNow().Add(3 * time.Second)
}

func (m *Model) startAnimation(animType AnimationType) {
	m.Animation = Animation{
		Type:      animType,
		Frame:     0,
		StartTime: pet.TimeNow(),
	}
}

func (m *Model) administerMedicine() bool {
	m.modifyStats(func(p *pet.Pet) {
		p.Illness = false
		bondMultiplier := p.GetBondMultiplier()
		healthGain := int(float64(pet.MedicineEffect) * bondMultiplier)
		p.Health = min(p.Health+healthGain, pet.MaxStat)
		p.AddInteraction("medicine")
		p.UpdateBond(pet.BondGainWellTimed)
		log.Printf("Administered medicine (bond mult: %.2f). Health is now %d", bondMultiplier, p.Health)
	})
	m.startAnimation(AnimMedicine)
	return true
}

func (m *Model) feed() bool {
	if m.Pet.Hunger >= 90 {
		m.setMessage("🍽️ Not hungry right now!")
		return false
	}

	recentFeeds := pet.CountRecentInteractions(m.Pet.LastInteractions, "feed", pet.SpamPreventionWindow)
	hungerBefore := m.Pet.Hunger

	m.modifyStats(func(p *pet.Pet) {
		p.Sleeping = false
		p.AutoSleepTime = nil
		p.FractionalEnergy = 0

		effectiveness := 1.0
		if recentFeeds > 0 {
			effectiveness = 1.0 / float64(recentFeeds+1)
		}

		bondMultiplier := p.GetBondMultiplier()
		hungerGain := int(float64(pet.FeedHungerIncrease) * p.GetTraitModifier("feed_bonus") * effectiveness * bondMultiplier)
		happinessGain := int(float64(pet.FeedHappinessIncrease) * p.GetTraitModifier("feed_bonus_happiness") * effectiveness * bondMultiplier)

		p.Hunger = min(p.Hunger+hungerGain, pet.MaxStat)
		p.Happiness = min(p.Happiness+happinessGain, pet.MaxStat)
		p.AddInteraction("feed")

		if recentFeeds == 0 && hungerBefore < 50 {
			p.UpdateBond(pet.BondGainWellTimed)
		} else if recentFeeds == 0 {
			p.UpdateBond(pet.BondGainNormal)
		}

		log.Printf("Fed pet (effectiveness: %.2f, bond mult: %.2f). Hunger is now %d, Happiness is now %d",
			effectiveness, bondMultiplier, p.Hunger, p.Happiness)
	})
	m.setMessage("🍖 Yum!")
	m.startAnimation(AnimFeed)
	return true
}

func (m *Model) play() bool {
	if m.Pet.Energy < pet.AutoSleepThreshold {
		m.setMessage("😴 Too tired to play...")
		return false
	}

	if m.Pet.Mood == "lazy" && m.Pet.Energy < 50 {
		m.setMessage("😪 Not in the mood to play...")
		return false
	}

	currentHour := pet.TimeNow().Local().Hour()
	isActive := pet.IsActiveHours(&m.Pet, currentHour)

	recentPlays := pet.CountRecentInteractions(m.Pet.LastInteractions, "play", pet.SpamPreventionWindow)
	happinessBefore := m.Pet.Happiness

	m.modifyStats(func(p *pet.Pet) {
		p.Sleeping = false
		p.AutoSleepTime = nil
		p.FractionalEnergy = 0

		effectiveness := 1.0
		if recentPlays > 0 {
			effectiveness = 1.0 / float64(recentPlays+1)
		}

		bondMultiplier := p.GetBondMultiplier()
		happinessGain := float64(pet.PlayHappinessIncrease)
		if !isActive {
			happinessGain *= pet.OutsideActiveHappinessMult
		}
		happinessGain *= p.GetTraitModifier("play_bonus")
		happinessGain *= bondMultiplier * effectiveness

		p.Happiness = min(p.Happiness+int(happinessGain), pet.MaxStat)
		p.Energy = max(p.Energy-pet.PlayEnergyDecrease, pet.MinStat)
		p.Hunger = max(p.Hunger-pet.PlayHungerDecrease, pet.MinStat)
		p.AddInteraction("play")

		if recentPlays == 0 && happinessBefore < 50 {
			p.UpdateBond(pet.BondGainWellTimed)
		} else if recentPlays == 0 {
			p.UpdateBond(pet.BondGainNormal)
		}

		log.Printf("Played with pet (effectiveness: %.2f, bond mult: %.2f). Happiness is now %d, Energy is now %d, Hunger is now %d",
			effectiveness, bondMultiplier, p.Happiness, p.Energy, p.Hunger)
	})

	if !isActive {
		m.setMessage("🥱 *yawn* ...play time...")
	} else if m.Pet.Mood == "playful" {
		m.setMessage("🎉 So much fun!")
	} else {
		m.setMessage("🎾 Wheee!")
	}
	m.startAnimation(AnimPlay)
	return true
}

func (m *Model) toggleSleep() bool {
	m.modifyStats(func(p *pet.Pet) {
		p.Sleeping = !p.Sleeping
		p.AutoSleepTime = nil
		p.FractionalEnergy = 0
		log.Printf("Pet is now sleeping: %t", p.Sleeping)
	})
	if m.Pet.Sleeping {
		m.startAnimation(AnimSleep)
	}
	return m.Pet.Sleeping
}

func (m *Model) updateHourlyStats(t time.Time) {
	m.modifyStats(func(p *pet.Pet) {
		if int(t.Minute()) == 0 {
			p.RecordStatCheckpoint()
		}

		if int(t.Minute()) == 0 {
			hungerRate := pet.HungerDecreaseRate
			if p.Sleeping {
				hungerRate = pet.SleepingHungerRate
			}
			p.Hunger = max(p.Hunger-hungerRate, pet.MinStat)
			log.Printf("Hunger decreased to %d", p.Hunger)
		}

		if !p.Sleeping {
			if int(t.Hour())%2 == 0 && int(t.Minute()) == 0 {
				p.Energy = max(p.Energy-pet.EnergyDecreaseRate, pet.MinStat)
				log.Printf("Energy decreased to %d", p.Energy)
			}
		} else {
			if int(t.Minute()) == 0 {
				p.Energy = min(p.Energy+pet.EnergyRecoveryRate, pet.MaxStat)
				log.Printf("Energy increased to %d", p.Energy)
			}
		}

		if p.Hunger < 30 || p.Energy < 30 {
			if int(t.Minute()) == 0 {
				p.Happiness = max(p.Happiness-2, 0)
				log.Printf("Happiness decreased to %d", p.Happiness)
			}
		}

		if p.Hunger < 15 || p.Happiness < 15 || p.Energy < 15 {
			if int(t.Minute()) == 0 {
				healthRate := 2
				if p.Sleeping {
					healthRate = 1
				}
				p.Health = max(p.Health-healthRate, pet.MinStat)
				log.Printf("Health decreased to %d", p.Health)
			}
		}
	})
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
