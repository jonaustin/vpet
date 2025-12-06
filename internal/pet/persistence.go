package pet

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

// TestConfigPath is used for testing to override the config path
var TestConfigPath string

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

// GetConfigPath returns the path to the pet state file
func GetConfigPath() string {
	if TestConfigPath != "" {
		return TestConfigPath
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

// NewPet creates a new pet with default values or test values if provided
func NewPet(testCfg *TestConfig) Pet {
	now := TimeNow()
	var p Pet
	var birthTime = now

	if testCfg != nil {
		birthTime = testCfg.LastSavedTime
		if birthTime.IsZero() {
			birthTime = now
		}
		p = Pet{
			Name:      DefaultPetName,
			Hunger:    testCfg.InitialHunger,
			Happiness: testCfg.InitialHappiness,
			Energy:    testCfg.InitialEnergy,
			Health:    testCfg.Health,
			Age:       0,
			LifeStage: 0,
			Sleeping:  testCfg.IsSleeping,
			LastSaved: birthTime,
			Illness:   testCfg.Illness,
		}
	} else {
		p = Pet{
			Name:      DefaultPetName,
			Hunger:    MaxStat,
			Happiness: MaxStat,
			Energy:    MaxStat,
			Health:    MaxStat,
			Age:       0,
			LifeStage: 0,
			Form:      FormBaby,
			Sleeping:  false,
			LastSaved: now,
			Illness:   false,
		}
	}

	// Initialize evolution tracking maps
	if p.Form == 0 {
		p.Form = FormBaby
	}
	if p.CareQualityHistory == nil {
		p.CareQualityHistory = make(map[int]CareQuality)
	}
	if p.StatCheckpoints == nil {
		p.StatCheckpoints = make(map[string][]StatCheck)
	}

	// Assign random chronotype at birth
	if p.Chronotype == "" {
		p.Chronotype = AssignRandomChronotype()
		log.Printf("Assigned chronotype: %s", GetChronotypeName(p.Chronotype))
	}

	// Assign random personality traits at birth
	if len(p.Traits) == 0 {
		p.Traits = GenerateTraits()
	}

	// Initialize bond for new pets
	if p.Bond == 0 {
		p.Bond = InitialBond
		log.Printf("Initialized bond at %d", InitialBond)
	}

	p.LastStatus = GetStatus(p)
	// Add initial log entry with birth time
	p.Logs = []LogEntry{{
		Time:      birthTime,
		OldStatus: "",
		NewStatus: p.LastStatus,
	}}
	log.Printf("Created new pet: %s", p.Name)
	return p
}

// LoadState loads the pet's state from file or creates a new pet
func LoadState() Pet {
	configPath := GetConfigPath()
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Printf("Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Error reading state file: %v. Creating new pet.", err)
		return NewPet(nil)
	}

	var p Pet
	if err := json.Unmarshal(data, &p); err != nil {
		log.Printf("Error loading state: %v. Creating new pet.", err)
		return NewPet(nil)
	}

	// Update stats based on elapsed time and check for death
	now := TimeNow()
	log.Printf("last saved: %s\n", p.LastSaved.UTC())
	elapsed := now.Sub(p.LastSaved.UTC())
	log.Printf("elapsed %f\n", elapsed.Seconds())
	elapsedHours := elapsed.Hours()

	// Store current status before updates
	oldStatus := p.LastStatus
	if oldStatus == "" {
		oldStatus = GetStatus(p)
	}

	// Update age and life stage
	birthTime := p.Logs[0].Time
	p.Age = int(now.Sub(birthTime).Hours())

	// Calculate life stage based on age and handle evolution
	oldLifeStage := p.LifeStage
	if p.Age < AgeStageThresholds {
		p.LifeStage = 0 // Baby
	} else if p.Age < 2*AgeStageThresholds {
		p.LifeStage = 1 // Child
	} else {
		p.LifeStage = 2 // Adult
	}

	// Handle evolution when life stage changes
	if oldLifeStage != p.LifeStage && p.LifeStage > 0 {
		p.Evolve(p.LifeStage)
	}

	// Check death condition first
	if p.Dead {
		return p
	}

	// Calculate hunger decrease with trait modifiers
	hungerRate := float64(HungerDecreaseRate)
	if p.Sleeping {
		hungerRate = float64(SleepingHungerRate)
	}
	hungerRate *= p.GetTraitModifier("hunger_decay")
	hungerLoss := int(elapsedHours * hungerRate)
	p.Hunger = max(p.Hunger-hungerLoss, MinStat)

	// Apply chronotype-based multipliers
	currentHour := now.Local().Hour()
	isActive := IsActiveHours(&p, currentHour)

	if !p.Sleeping {
		// Energy decreases when awake
		energyMult := 1.0
		if !isActive {
			energyMult = OutsideActiveEnergyMult
		}
		energyMult *= p.GetTraitModifier("energy_decay")
		energyLoss := int((elapsedHours / 2.0) * float64(EnergyDecreaseRate) * energyMult)
		p.Energy = max(p.Energy-energyLoss, MinStat)
	} else {
		// Energy recovers while sleeping
		recoveryMult := 1.0
		if !isActive {
			recoveryMult = PreferredSleepRecoveryMult
		}
		exactGain := elapsedHours * float64(EnergyRecoveryRate) * recoveryMult
		p.FractionalEnergy += exactGain
		wholeGain := int(p.FractionalEnergy)
		p.FractionalEnergy -= float64(wholeGain)
		p.Energy = min(p.Energy+wholeGain, MaxStat)
	}

	// Update happiness if stats are low
	if p.Hunger < LowStatThreshold || p.Energy < LowStatThreshold {
		happinessRate := float64(HappinessDecreaseRate) * p.GetTraitModifier("happiness_decay")
		happinessLoss := int(elapsedHours * happinessRate)
		p.Happiness = max(p.Happiness-happinessLoss, MinStat)
	}

	// Update bond from neglect
	if len(p.LastInteractions) > 0 {
		var mostRecent = p.LastInteractions[0].Time
		for _, interaction := range p.LastInteractions {
			if interaction.Time.After(mostRecent) {
				mostRecent = interaction.Time
			}
		}
		hoursSinceInteraction := now.Sub(mostRecent).Hours()

		if hoursSinceInteraction > BondDecayThreshold {
			excessHours := hoursSinceInteraction - BondDecayThreshold
			bondLoss := int(excessHours/12) * BondDecayRate
			if bondLoss > 0 {
				p.Bond = max(p.Bond-bondLoss, 0)
				log.Printf("Bond decreased by %d from neglect (%.1f hours since last interaction)", bondLoss, hoursSinceInteraction)
			}
		}
	}

	// Check for random illness when health is low
	if p.Health < 50 && !p.Illness {
		adjustedIllnessChance := IllnessChance * p.GetTraitModifier("illness_chance")
		if p.Bond >= IllnessResistanceBond {
			bondReduction := 1.0 - (float64(p.Bond-IllnessResistanceBond) / float64(MaxBond-IllnessResistanceBond) * 0.5)
			adjustedIllnessChance *= bondReduction
		}
		if RandFloat64() < adjustedIllnessChance {
			p.Illness = true
		}
	} else if p.Health >= 50 {
		p.Illness = false
	}

	// Health decreases when any stat is critically low
	if p.Hunger < 15 || p.Happiness < 15 || p.Energy < 15 {
		healthRate := 2.0
		if p.Sleeping {
			healthRate = 1.0
		}
		healthRate *= p.GetTraitModifier("health_decay")
		healthLoss := int(elapsedHours * healthRate)
		p.Health = max(p.Health-healthLoss, MinStat)
	}

	// Check if any critical stat is below threshold
	inCriticalState := p.Health <= 20 || p.Hunger < 10 ||
		p.Happiness < 10 || p.Energy < 10

	// Track time in critical state
	if inCriticalState {
		if p.CriticalStartTime == nil {
			p.CriticalStartTime = &now
		}

		if now.Sub(*p.CriticalStartTime) > DeathTimeThreshold {
			p.Dead = true
			p.CauseOfDeath = "Neglect"

			if p.Hunger <= 0 {
				p.CauseOfDeath = "Starvation"
			} else if p.Illness {
				p.CauseOfDeath = "Sickness"
			}
		}
	} else {
		p.CriticalStartTime = nil
	}

	// Check for natural death from old age
	if p.Age >= MinNaturalLifespan && RandFloat64() < float64(p.Age-MinNaturalLifespan)/1000 {
		p.Dead = true
		p.CauseOfDeath = "Old Age"
	}

	// Apply autonomous behavior
	if !p.Dead {
		ApplyAutonomousBehavior(&p)
	}

	// Trigger random life events
	TriggerRandomEvent(&p)

	p.LastSaved = now
	return p
}

// SaveState saves the pet's state to file
func SaveState(p *Pet) {
	now := TimeNow()
	birthTime := p.Logs[0].Time
	p.Age = int(now.Sub(birthTime).Hours())
	p.LastSaved = now

	currentStatus := GetStatus(*p)
	if p.LastStatus == "" {
		p.LastStatus = currentStatus
	}

	if currentStatus != p.LastStatus {
		if p.Logs == nil {
			p.Logs = []LogEntry{}
		}

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
	if err := os.WriteFile(GetConfigPath(), data, 0644); err != nil {
		log.Printf("Error writing state: %v", err)
	}
}

// ApplyAutonomousBehavior makes the pet act on its own based on current state
func ApplyAutonomousBehavior(p *Pet) {
	now := TimeNow()
	currentHour := now.Local().Hour()
	isActive := IsActiveHours(p, currentHour)

	// Determine auto-sleep threshold based on chronotype
	sleepThreshold := AutoSleepThreshold
	if !isActive {
		sleepThreshold = DrowsyThreshold
	}

	// Auto-sleep when exhausted
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

		shouldWake := false
		if sleepHours >= MaxSleepDuration {
			shouldWake = true
		} else if sleepHours >= MinSleepDuration && p.Energy >= AutoWakeEnergy && isActive {
			shouldWake = true
		}

		if shouldWake {
			p.Sleeping = false
			p.AutoSleepTime = nil
			p.FractionalEnergy = 0
			log.Printf("Pet woke up automatically after %.1f hours of sleep (Energy: %d)", sleepHours, p.Energy)
		}
	}

	// Random mood changes
	if p.Mood == "" {
		p.Mood = "normal"
	}
	if p.MoodExpiresAt == nil || now.After(*p.MoodExpiresAt) {
		var newMood string
		roll := RandFloat64()

		if p.Energy < DrowsyThreshold {
			if roll < 0.6 {
				newMood = "lazy"
			} else if roll < 0.8 {
				newMood = "needy"
			} else {
				newMood = "normal"
			}
		} else if p.Happiness < BoredThreshold {
			if roll < 0.5 {
				newMood = "needy"
			} else if roll < 0.7 {
				newMood = "playful"
			} else {
				newMood = "normal"
			}
		} else if p.Hunger < HungryThreshold {
			if roll < 0.5 {
				newMood = "needy"
			} else {
				newMood = "normal"
			}
		} else {
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
		moodDuration := (2 + int(RandFloat64()*2)) * int(1)
		expires := now.Add(time.Duration(moodDuration) * time.Hour)
		p.MoodExpiresAt = &expires
		log.Printf("Pet mood changed to: %s (expires in %d hours)", newMood, moodDuration)
	}
}
