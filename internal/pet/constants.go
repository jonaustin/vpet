package pet

import "time"

// Game constants
const (
	DefaultPetName     = "Charm Pet"
	MaxStat            = 100
	MinStat            = 0
	LowStatThreshold   = 30
	DeathTimeThreshold = 12 * time.Hour // Time in critical state before death
	HealthDecreaseRate = 2              // Health loss per hour
	AgeStageThresholds = 48             // Hours per life stage
	IllnessChance      = 0.1            // 10% chance per hour when health <50
	MedicineEffect     = 30             // Health restored by medicine
	MinNaturalLifespan = 168            // Hours before natural death possible (~1 week)

	// Stat change rates (per hour)
	HungerDecreaseRate    = 5
	SleepingHungerRate    = 3 // 70% of normal rate
	EnergyDecreaseRate    = 5
	EnergyRecoveryRate    = 10
	HappinessDecreaseRate = 2

	FeedHungerIncrease    = 30
	FeedHappinessIncrease = 10
	PlayHappinessIncrease = 30
	PlayEnergyDecrease    = 10
	PlayHungerDecrease    = 5

	// Care quality thresholds for evolution
	PerfectCareThreshold = 85
	GoodCareThreshold    = 70
	PoorCareThreshold    = 40
	NeglectThreshold     = 20

	// High stat thresholds
	HighStatThreshold    = 80 // Threshold for "very high" stats (used in chase mode for very happy emoji)

	// Autonomous behavior thresholds
	AutoSleepThreshold  = 20 // Energy level that triggers auto-sleep
	DrowsyThreshold     = 40 // Energy level that shows drowsy status
	WantHungerThreshold = 40 // Hunger deficit to show 🍖 want (Hunger <= 60)
	WantHappyThreshold  = 40 // Happiness deficit to show 🎾 want (Happiness <= 60)
	WantEnergyThreshold = 55 // Energy deficit to show 🛌 want (Energy <= 45)
	AutoWakeEnergy      = 80 // Energy level to wake up automatically
	MinSleepDuration    = 6  // Minimum hours of auto-sleep
	MaxSleepDuration    = 8  // Maximum hours before forced wake
	HungryThreshold     = 30 // Hunger level to show "wants food"
	BoredThreshold      = 30 // Happiness level to show "wants play"

	// Chronotype multipliers
	OutsideActiveEnergyMult    = 1.5 // 50% faster energy drain outside active hours
	OutsideActiveHappinessMult = 0.7 // 30% less happiness gain outside active hours
	PreferredSleepRecoveryMult = 1.2 // 20% better sleep recovery during preferred hours

	// Bonding system constants
	MaxBond               = 100           // Maximum bond level
	InitialBond           = 50            // Starting bond for new pets
	BondDecayThreshold    = 24            // Hours before bond starts decaying from neglect
	BondDecayRate         = 1             // Bond lost per 12 hours of neglect beyond threshold
	SpamPreventionWindow  = 1 * time.Hour // Time window to check for repeated actions
	MinBondMultiplier     = 0.5           // Action effectiveness at 0 bond
	MaxBondMultiplier     = 1.0           // Action effectiveness at 100 bond
	BondGainWellTimed     = 2             // Bond gained for well-timed action
	BondGainNormal        = 1             // Bond gained for normal action
	IllnessResistanceBond = 70            // Bond level that starts reducing illness chance
	MaxInteractionHistory = 20            // Keep last 20 interactions

	// Status emojis
	StatusEmojiHappy       = "😸" // Default happy status
	StatusEmojiNeutral     = "🙂" // Neutral/normal state
	StatusEmojiSleeping    = "😴" // Sleeping/tired
	StatusEmojiHungry      = "🙀" // Hungry/desperate
	StatusEmojiSad         = "😿" // Sad/unhappy
	StatusEmojiEnergetic   = "😼" // Energetic/fast
	StatusEmojiExcited     = "😻" // Excited/about to catch
	StatusEmojiSick        = "🤢" // Sick/ill
	StatusEmojiTired       = "😾" // Tired/grumpy
	StatusEmojiDead        = "💀" // Dead
)

// Chronotype constants
const (
	ChronotypeEarlyBird = "early_bird" // 5am-9pm active
	ChronotypeNormal    = "normal"     // 7am-11pm active
	ChronotypeNightOwl  = "night_owl"  // 10am-2am active
)

// PetForm represents evolution forms
type PetForm int

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
