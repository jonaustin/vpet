# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

vpet is a Tamagotchi-style virtual pet that lives in your terminal with tmux integration. It's a single-file Go application using the Bubble Tea TUI framework. The pet has lifecycle mechanics with aging, stats that decay over time, random illnesses, and multiple death conditions.

## Development Commands

### Building and Running
```bash
# Build the binary
go build -o vpet main.go

# Run interactive mode
go run main.go

# Update stats without UI (used by tmux integration)
go run main.go -u

# Get status emoji only
go run main.go -status
```

### Testing
```bash
# Run all tests
go test -v

# Run specific test
go test -v -run TestName

# Run tests with coverage
go test -v -cover
```

## Architecture

### Single-File Structure
All code lives in `main.go`. The application is intentionally kept simple with no package separation.

### Core Components

**Pet State (`Pet` struct)**
- Persists to `~/.config/vpet/pet.json`
- Tracks stats (Hunger, Happiness, Energy, Health), age, life stage, illness state
- Records status transition logs for history tracking
- Uses UTC timestamps throughout for consistency

**Game Loop (`model` struct)**
- Implements Bubble Tea's `tea.Model` interface
- Handles keyboard input, stat updates, and rendering
- Modifies stats through `modifyStats()` helper which saves immediately after each change

**Time-Based Updates**
- Stats decay based on elapsed time since `LastSaved`
- Calculated in `loadState()` when reopening the app
- `updateHourlyStats()` handles minute-by-minute decay during active sessions
- **Critical**: All time operations use UTC via `timeNow()` variable (mockable for testing)

### State Persistence Pattern

The app follows an immediate-save pattern:
1. User action (feed, play, etc.) calls a method like `feed()`
2. Method calls `modifyStats(func(*Pet))` with stat changes
3. `modifyStats()` applies changes and immediately calls `saveState()`
4. `saveState()` updates `LastSaved`, calculates age from birth time in logs, tracks status transitions, and writes to JSON

**Status Logging System**
- Every `saveState()` call checks if pet's emoji status changed
- If changed, appends `LogEntry{Time, OldStatus, NewStatus}` to `Pet.Logs`
- Birth time is recorded as first log entry's timestamp
- Age is calculated from birth time on each save: `now.Sub(birthTime).Hours()`

### Testing Strategy

ALWAYS add/update tests for new logic.

**Time Mocking**
- `timeNow` variable allows test control of current time
- `randFloat64` variable allows deterministic illness testing
- Tests set exact timestamps and verify elapsed time calculations

**File Isolation**
- `testConfigPath` variable overrides config location
- `setupTestFile(t)` creates temp directory, returns cleanup function
- Tests manually manipulate JSON files to set exact `LastSaved` times

**Test Pattern for Time-Based Logic**
```go
// 1. Set fixed current time
originalTimeNow := timeNow
timeNow = func() time.Time { return fixedTime }
defer func() { timeNow = originalTimeNow }()

// 2. Create pet with past LastSaved
pet := newPet(&TestConfig{LastSavedTime: pastTime})
saveState(&pet)

// 3. Manually fix LastSaved in JSON (saveState overwrites it)
data, _ := os.ReadFile(testConfigPath)
json.Unmarshal(data, &savedPet)
savedPet.LastSaved = pastTime
json.MarshalIndent(savedPet, "", "  ")
os.WriteFile(testConfigPath, data, 0644)

// 4. Load and verify calculations
loadedPet := loadState()
// Assert expected stat changes
```

### Key Mechanics

**Aging and Life Stages**
- Age increases based on elapsed hours in `loadState()` (line 312)
- Life stages determined by age thresholds: 0-24h Baby, 24-48h Child, 48h+ Adult (lines 315-321)
- Age calculation in `saveState()` uses birth time from first log entry (lines 408-411)

**Critical State and Death**
- Pet enters critical state when Health ≤20, Hunger <10, Happiness <10, or Energy <10
- `CriticalStartTime` tracks when critical state began
- Death occurs after 4 hours in critical state (`deathTimeThreshold`)
- Natural death possible after 72 hours with increasing probability
- Death causes: Neglect, Starvation, Sickness, Old Age

**Illness System**
- Random illness when Health <50 (10% chance per hour in `loadState()`)
- Medicine cures illness and restores +30 Health
- Illness auto-clears when Health ≥50

**Stat Decay Rates** (defined in constants)
- Hunger: -5%/hr awake, -3%/hr sleeping
- Energy: -5% per 2hrs awake, +10%/hr sleeping
- Health: -2%/hr when any stat <30
- Happiness: -2%/hr when Hunger or Energy <30

**Tmux Integration**
- `vpet.tmux` script runs `vpet -u` every 5 seconds in background
- Updates tmux `status-left` with emoji from `vpet -status`
- PID tracked at `~/.config/vpet/tmux_update.pid`
- **Note**: Requires updating `VPET_DIR` variable to match installation path

### Important Code Locations

- Constants and game configuration: lines 27-51
- Pet state structure: lines 54-70
- Stats modification pattern: lines 79-126
- Time-based stat decay: lines 128-165, 328-400
- State persistence: lines 275-443
- Bubble Tea UI: lines 458-601
- Status determination: lines 603-620
