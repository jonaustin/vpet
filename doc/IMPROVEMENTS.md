# vpet Improvement Ideas

## Current State Summary

The pet simulation has solid foundations: lifecycle stages, stat interdependencies, evolution based on care quality, and death conditions. However, the pet is essentially **passive** - it just decays until you interact with it.

---

## 5 Ideas to Enhance Realism (Ranked by Impact)

### 1. Autonomous Pet Behavior System ⭐ HIGHEST IMPACT

**The Problem:** Pet never acts on its own - it just waits for you to do something.

**The Solution:** Give the pet agency:
- **Auto-sleep** when Energy < 20 (drowsy warning at < 40)
- **Auto-wake** after 6-8 hours of sleep when Energy is restored
- **Refuse actions** - won't play when too tired, won't eat when full
- **Random moods** - "playful" (wants play), "lazy" (resists activity), "needy" (seeks attention)
- **Attention seeking** - status emoji shows what pet wants (🍖 hungry, 🎾 wants play)

**Why it matters:** This single change transforms the pet from a passive stat-tracker into something that feels alive.

**Complexity:** Medium

**Implementation approach:**
```go
// Add new fields to Pet struct
type Pet struct {
    // ... existing fields
    Mood          string     `json:"mood"`           // "normal", "playful", "lazy", "needy"
    MoodExpiresAt *time.Time `json:"mood_expires_at"`
    AutoSleepTime *time.Time `json:"auto_sleep_time"` // When pet fell asleep on its own
}

// In loadState(), add autonomous behavior checks
func applyAutonomousBehavior(p *Pet) {
    // Auto-sleep when exhausted
    if p.Energy < 20 && !p.Sleeping {
        p.Sleeping = true
        now := timeNow()
        p.AutoSleepTime = &now
    }

    // Auto-wake after 6-8 hours of sleep
    if p.Sleeping && p.AutoSleepTime != nil {
        sleepDuration := timeNow().Sub(*p.AutoSleepTime)
        if sleepDuration >= 6*time.Hour && p.Energy >= 80 {
            p.Sleeping = false
            p.AutoSleepTime = nil
        }
    }

    // Random mood changes
    if p.MoodExpiresAt == nil || timeNow().After(*p.MoodExpiresAt) {
        moods := []string{"normal", "playful", "lazy", "needy"}
        weights := []float64{0.7, 0.1, 0.1, 0.1}
        p.Mood = weightedRandom(moods, weights)
        // Mood lasts 2-4 hours
        expires := timeNow().Add(time.Duration(2+rand.Intn(2)) * time.Hour)
        p.MoodExpiresAt = &expires
    }
}
```

---

### 2. Circadian Rhythm / Day-Night Cycle

**The Problem:** Time is meaningless - 3am and 3pm are identical to the pet.

**The Solution:** Add time-aware behavior:
- Pet has preferred wake/sleep hours (e.g., 7am-10pm)
- Energy decays **faster** if awake during "sleep hours"
- Happiness bonus for interacting during pet's active hours
- Pet types: "Early Bird" vs "Night Owl" affect schedules
- Weekend bonus? (Pet happier when you're around more)

**Why it matters:** Real pets have biological clocks. A dog waiting for you at 6am to go outside is realistic.

**Complexity:** Medium

**Implementation approach:**
```go
// Add to Pet struct
type Pet struct {
    // ... existing
    PreferredWakeHour  int `json:"preferred_wake_hour"`  // e.g., 6
    PreferredSleepHour int `json:"preferred_sleep_hour"` // e.g., 22
    Chronotype         string `json:"chronotype"` // "early_bird", "night_owl", "normal"
}

// Add time-aware decay multipliers
func getTimeMultiplier(hour int, pet *Pet) float64 {
    // Pet is more active during their "awake" window
    if hour >= pet.PreferredWakeHour && hour < pet.PreferredSleepHour {
        return 1.0 // Normal decay
    }
    // Outside preferred hours
    if !pet.Sleeping {
        return 1.5 // 50% faster energy decay when awake at "wrong" time
    }
    return 0.8 // Better sleep during preferred sleep hours
}

// Modify happiness bonus for timely interactions
func (m *model) play() {
    hour := timeNow().Hour()
    bonus := playHappinessIncrease
    if isActiveHours(hour, &m.pet) {
        bonus = int(float64(bonus) * 1.2) // 20% bonus during active hours
    }
    // ... rest of play logic
}
```

---

### 3. Personality Trait System

**The Problem:** Every pet behaves identically at the same stat levels.

**The Solution:** Random traits assigned at birth:
- **Temperament:** Calm (slower decay) vs Hyperactive (faster energy drain, more play rewards)
- **Appetite:** Picky (smaller feed bonus) vs Hungry (larger bonus, faster hunger decay)
- **Sociability:** Independent (slower happiness decay) vs Needy (bonus from frequent care)
- **Constitution:** Robust (resists illness) vs Fragile (gets sick easier)

**Why it matters:** Every pet becomes unique. Players adapt care strategies to their pet's personality. Creates replayability.

**Complexity:** Medium

**Implementation approach:**
```go
type Trait struct {
    Name        string             `json:"name"`
    Modifiers   map[string]float64 `json:"modifiers"` // stat_name -> multiplier
}

type Pet struct {
    // ... existing
    Traits []Trait `json:"traits"`
}

func generateTraits() []Trait {
    possibleTraits := map[string][]Trait{
        "temperament": {
            {Name: "Calm", Modifiers: map[string]float64{"energy_decay": 0.8, "happiness_decay": 0.9}},
            {Name: "Hyperactive", Modifiers: map[string]float64{"energy_decay": 1.3, "play_bonus": 1.5}},
        },
        "appetite": {
            {Name: "Picky", Modifiers: map[string]float64{"feed_bonus": 0.7}},
            {Name: "Hungry", Modifiers: map[string]float64{"hunger_decay": 1.2, "feed_bonus": 1.2}},
        },
        // ... more categories
    }

    var traits []Trait
    for _, options := range possibleTraits {
        traits = append(traits, options[rand.Intn(len(options))])
    }
    return traits
}

// Apply trait modifiers in stat calculations
func getEffectiveDecayRate(baseRate int, statType string, pet *Pet) int {
    multiplier := 1.0
    for _, trait := range pet.Traits {
        if mod, ok := trait.Modifiers[statType+"_decay"]; ok {
            multiplier *= mod
        }
    }
    return int(float64(baseRate) * multiplier)
}
```

---

### 4. Bonding/Trust Meter with Interaction Memory

**The Problem:** Spam-feeding works just as well as consistent care. No long-term relationship.

**The Solution:** Hidden "Bond" stat (0-100):
- **Grows** from consistent, well-timed care (feeding when hungry, not spamming)
- **Decays** from neglect (24+ hours without interaction)
- **High bond benefits:** Faster recovery, less illness, pet is "easier"
- **Low bond penalties:** Actions less effective, pet becomes "distant"
- **Interaction memory:** Track recent actions, diminishing returns for spam

**Why it matters:** Rewards being a good pet owner over time, not just keeping stats green.

**Complexity:** Medium-High

**Implementation approach:**
```go
type Pet struct {
    // ... existing
    Bond             int           `json:"bond"` // 0-100
    LastInteractions []Interaction `json:"last_interactions"`
}

type Interaction struct {
    Type string    `json:"type"` // "feed", "play", "medicine"
    Time time.Time `json:"time"`
}

func (m *model) feed() {
    // Check for recent feeds (spam prevention)
    recentFeeds := countRecentInteractions(m.pet.LastInteractions, "feed", 1*time.Hour)

    effectiveness := 1.0
    if recentFeeds > 0 {
        effectiveness = 1.0 / float64(recentFeeds+1) // Diminishing returns
    }

    // Bond bonus
    bondMultiplier := 0.5 + (float64(m.pet.Bond) / 200.0) // 0.5x to 1.0x based on bond

    hungerGain := int(float64(feedHungerIncrease) * effectiveness * bondMultiplier)

    m.modifyStats(func(p *Pet) {
        p.Hunger = min(p.Hunger+hungerGain, maxStat)
        p.LastInteractions = append(p.LastInteractions, Interaction{"feed", timeNow()})

        // Increase bond for well-timed feeding (not spam)
        if recentFeeds == 0 && p.Hunger < 50 {
            p.Bond = min(p.Bond+2, 100)
        }
    })
}

// Bond decay from neglect (called in loadState)
func updateBond(p *Pet, elapsedHours float64) {
    if elapsedHours > 24 {
        bondLoss := int((elapsedHours - 24) / 12) // -1 bond per 12 hours of neglect
        p.Bond = max(p.Bond-bondLoss, 0)
    }
}
```

---

### 5. Expanded Interaction Types with Mini-Activities

**The Problem:** "Feed" and "Play" are generic one-size-fits-all actions.

**The Solution:** Multiple activity options with trade-offs:

**Feeding:**
| Option | Effect | Trade-off |
|--------|--------|-----------|
| Quick Snack | +15 Hunger | No cooldown |
| Full Meal | +40 Hunger, +5 Happiness | 4hr cooldown |
| Treat | +5 Hunger, +20 Happiness | -2 Health if overused |

**Play:**
| Option | Effect | Trade-off |
|--------|--------|-----------|
| Quick Pet | +10 Happiness, +5 Bond | Minimal |
| Active Play | +30 Happiness | -15 Energy, -10 Hunger |
| Training | +15 Happiness, +10 Bond | -10 Energy, teaches tricks |
| Rest Together | +10 Happiness, +5 Energy | Takes time |

**Why it matters:** Creates meaningful choices. Do you give a treat (happy now, unhealthy later) or proper food?

**Complexity:** High (requires menu/UI changes)

**Implementation approach:**
```go
type Activity struct {
    Name       string
    Effects    map[string]int // stat -> change amount
    Cooldown   time.Duration
    Conditions func(*Pet) bool // Can this activity be done?
}

var feedingOptions = []Activity{
    {
        Name:    "Quick Snack",
        Effects: map[string]int{"Hunger": 15},
        Cooldown: 0,
    },
    {
        Name:    "Full Meal",
        Effects: map[string]int{"Hunger": 40, "Happiness": 5},
        Cooldown: 4 * time.Hour,
        Conditions: func(p *Pet) bool {
            return timeSinceLastActivity(p, "Full Meal") > 4*time.Hour
        },
    },
    {
        Name:    "Treat",
        Effects: map[string]int{"Hunger": 5, "Happiness": 20, "Health": -2},
        Cooldown: 2 * time.Hour,
    },
}

// Update the menu to show submenus
func (m model) renderMenu() string {
    if m.inSubmenu {
        return renderActivitySubmenu(m.submenuType, m.subChoice)
    }
    // ... main menu
}
```

---

## Summary Ranking

| Rank | Idea | Impact on Realism | Complexity |
|------|------|------------------|------------|
| 1 | Autonomous Behavior | Very High | Medium |
| 2 | Circadian Rhythm | High | Medium |
| 3 | Personality Traits | High | Medium |
| 4 | Bonding System | Medium-High | Medium-High |
| 5 | Expanded Interactions | Medium | High |

---

## Quick Wins (Minimal Code)

If you want to start small:

1. **Auto-sleep** (~10 lines): Pet sleeps when Energy < 20
2. **Feeding cooldown** (~15 lines): "Not hungry yet" if fed within last hour
3. **Random mood** (~20 lines): Mood field affects status emoji randomly
