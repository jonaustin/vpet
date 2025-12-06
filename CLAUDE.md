# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

vpet is a Tamagotchi-style virtual pet that lives in your terminal with tmux integration. It uses the Bubble Tea TUI framework. The pet has lifecycle mechanics with aging, stats that decay over time, random illnesses, and multiple death conditions.

## Development Commands

### Building and Running
```bash
# Build the binary
go build -o vpet .

# Run interactive mode
go run .

# Update stats without UI (used by tmux integration)
go run . -u

# Get status emoji only
go run . -status

# Display detailed stats
go run . -stats

# Watch pet chase a butterfly
go run . -chase
```

### Testing
```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test
go test -v ./internal/pet -run TestName

# Run tests with coverage
go test -v -cover ./...
```

## Architecture

### Package Structure

```
vpet/
├── main.go                      # Entry point, flags, wiring
├── internal/
│   ├── pet/
│   │   ├── pet.go               # Pet struct, stats, lifecycle, evolution
│   │   ├── constants.go         # Decay rates, thresholds, form definitions
│   │   ├── persistence.go       # Save/load JSON state, autonomous behavior
│   │   ├── events.go            # Random events system
│   │   ├── status.go            # GetStatus functions
│   │   └── pet_test.go          # All pet-related tests
│   ├── chase/
│   │   └── chase.go             # Butterfly chase animation
│   └── ui/
│       ├── model.go             # Bubble Tea model, Update(), actions
│       ├── view.go              # Rendering, cheat menu
│       └── stats.go             # Stats display screen
```

### Core Packages

**pet/** - Core game logic
- `Pet` struct with all state fields
- `TimeNow` and `RandFloat64` vars for testing (mockable)
- Lifecycle: aging, evolution, death
- Traits, chronotypes, bond system
- Persists to `~/.config/vpet/pet.json`

**ui/** - Terminal UI
- Bubble Tea `Model` implementation
- User actions: feed, play, sleep, medicine
- Rendering and cheat menu
- Stats display screen

**chase/** - Mini animation
- Butterfly chase animation using Bubble Tea
- Pet follows butterfly with 2D movement

### Key Types and Functions

**pet/pet.go**
- `Pet` struct - all pet state
- `TimeNow`, `RandFloat64` - mockable for tests
- Evolution, traits, bond methods

**pet/persistence.go**
- `NewPet(*TestConfig)` - create pet
- `LoadState()` - load from disk + apply time decay
- `SaveState(*Pet)` - save to disk
- `ApplyAutonomousBehavior(*Pet)` - auto-sleep/wake, mood

**pet/events.go**
- `TriggerRandomEvent(*Pet)` - random event system
- `RespondToEvent()` - handle event interaction

**ui/model.go**
- `Model` struct - UI state
- `feed()`, `play()`, `toggleSleep()`, `administerMedicine()`
- `modifyStats(func(*Pet))` - modify + save pattern

### State Persistence Pattern

The app follows an immediate-save pattern:
1. User action (feed, play, etc.) calls a method like `feed()`
2. Method calls `modifyStats(func(*pet.Pet))` with stat changes
3. `modifyStats()` applies changes and immediately calls `pet.SaveState()`
4. `SaveState()` updates `LastSaved`, calculates age, tracks status transitions, writes JSON

### Testing Strategy

ALWAYS add/update tests for new logic.

**Time Mocking**
- `pet.TimeNow` variable allows test control of current time
- `pet.RandFloat64` variable allows deterministic illness testing
- Tests set exact timestamps and verify elapsed time calculations

**File Isolation**
- `pet.TestConfigPath` variable overrides config location
- `setupTestFile(t)` creates temp directory, returns cleanup function

**Test Helper**
- `testModel` struct in pet_test.go replicates UI model behavior
- Avoids circular dependency (pet tests can't import ui)

**Test Pattern for Time-Based Logic**
```go
// 1. Set fixed current time
originalTimeNow := pet.TimeNow
pet.TimeNow = func() time.Time { return fixedTime }
defer func() { pet.TimeNow = originalTimeNow }()

// 2. Create pet with past LastSaved
p := pet.NewPet(&pet.TestConfig{LastSavedTime: pastTime})
pet.SaveState(&p)

// 3. Manually fix LastSaved in JSON (SaveState overwrites it)
// ... manipulate JSON file ...

// 4. Load and verify calculations
loadedPet := pet.LoadState()
```

### Key Mechanics

**Aging and Life Stages**
- Age increases based on elapsed hours in `LoadState()`
- Life stages: Baby (0-24h), Child (24-48h), Adult (48h+)
- Evolution happens when life stage changes

**Critical State and Death**
- Critical when Health ≤20, Hunger <10, Happiness <10, or Energy <10
- `CriticalStartTime` tracks when critical state began
- Death after 4 hours in critical state (`DeathTimeThreshold`)
- Natural death possible after 72 hours with increasing probability
- Causes: Neglect, Starvation, Sickness, Old Age

**Illness System**
- Random illness when Health <50 (10% chance per hour)
- Medicine cures illness and restores +30 Health
- Auto-clears when Health ≥50

**Stat Decay Rates** (in pet/constants.go)
- Hunger: -5%/hr awake, -3%/hr sleeping
- Energy: -5% per 2hrs awake, +10%/hr sleeping
- Health: -2%/hr when any stat <30
- Happiness: -2%/hr when Hunger or Energy <30

**Tmux Integration**
- `vpet.tmux` script runs `vpet -u` every 5 seconds
- Updates tmux `status-left` with emoji from `vpet -status`
