package chase

import (
	"math"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"vpet/internal/pet"
)

func TestTargets(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		wantName  string
		wantEmoji string
	}{
		{
			name:      "butterfly exists",
			target:    "butterfly",
			wantName:  "butterfly",
			wantEmoji: "🦋",
		},
		{
			name:      "ball exists",
			target:    "ball",
			wantName:  "ball",
			wantEmoji: "⚽",
		},
		{
			name:      "mouse exists",
			target:    "mouse",
			wantName:  "mouse",
			wantEmoji: "🐁",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, exists := Targets[tt.target]
			if !exists {
				t.Fatalf("Target %q does not exist", tt.target)
			}
			if target.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", target.Name, tt.wantName)
			}
			if target.Emoji != tt.wantEmoji {
				t.Errorf("Emoji = %q, want %q", target.Emoji, tt.wantEmoji)
			}
			if target.Speed <= 0 {
				t.Errorf("Speed = %f, want > 0", target.Speed)
			}
		})
	}
}

func TestModel_Init(t *testing.T) {
	m := Model{
		Pet:        pet.Pet{},
		Target:     Targets["butterfly"],
		TermWidth:  80,
		TermHeight: 24,
	}

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() returned nil command, expected batch command")
	}
}

func TestModel_Update_KeyMsg(t *testing.T) {
	baseTime := time.Now()
	m := Model{
		Pet:            pet.Pet{},
		Target:         Targets["butterfly"],
		TermWidth:      80,
		TermHeight:     24,
		LastUpdateTime: baseTime,
		ElapsedTime:    1.5,
	}

	// Any key should quit
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// ElapsedTime should not change on key press
	if updatedModel.(Model).ElapsedTime != 1.5 {
		t.Error("KeyMsg should not modify model state")
	}

	// Check that quit command was returned
	if cmd == nil {
		t.Error("KeyMsg should return tea.Quit command")
	}
}

func TestModel_Update_WindowSizeMsg(t *testing.T) {
	m := Model{
		Pet:        pet.Pet{},
		Target:     Targets["butterfly"],
		TermWidth:  80,
		TermHeight: 24,
	}

	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 30,
	}

	updatedModel, _ := m.Update(msg)
	updated := updatedModel.(Model)

	if updated.TermWidth != 100 {
		t.Errorf("TermWidth = %d, want 100", updated.TermWidth)
	}
	if updated.TermHeight != 30 {
		t.Errorf("TermHeight = %d, want 30", updated.TermHeight)
	}
}

func TestModel_Update_AnimTick_ElapsedTimeIncrement(t *testing.T) {
	baseTime := time.Now()
	m := Model{
		Pet:            pet.Pet{},
		Target:         Targets["butterfly"],
		TermWidth:      80,
		TermHeight:     24,
		LastUpdateTime: baseTime,
		ElapsedTime:    0,
		PetPosX:        0,
		PetPosY:        12,
		TargetPosX:     5,
		TargetPosY:     12,
	}

	// Simulate 70ms tick
	nextTime := baseTime.Add(70 * time.Millisecond)
	updatedModel, _ := m.Update(animTickMsg(nextTime))
	updated := updatedModel.(Model)

	// ElapsedTime should have increased by ~0.07 seconds
	expectedElapsed := 0.07
	if updated.ElapsedTime < expectedElapsed-0.001 || updated.ElapsedTime > expectedElapsed+0.001 {
		t.Errorf("ElapsedTime = %f, want ~%f", updated.ElapsedTime, expectedElapsed)
	}
}

func TestModel_Update_AnimTick_TargetMovement(t *testing.T) {
	target := Targets["butterfly"]
	baseTime := time.Now()
	m := Model{
		Pet:            pet.Pet{},
		Target:         target,
		TermWidth:      80,
		TermHeight:     24,
		LastUpdateTime: baseTime,
		TargetPosX:     5,
		TargetPosY:     12,
	}

	// Simulate one tick (70ms)
	nextTime := baseTime.Add(70 * time.Millisecond)
	updatedModel, cmd := m.Update(animTickMsg(nextTime))
	updated := updatedModel.(Model)

	if cmd == nil {
		t.Error("animTickMsg should return tick command")
	}

	// Target should have moved horizontally
	// Butterfly speed is 8.0 columns/sec, so in 0.07 sec: 8.0 * 0.07 = 0.56 columns
	if updated.TargetPosX <= 5 {
		t.Errorf("TargetPosX = %f, expected > 5 after tick", updated.TargetPosX)
	}

	expectedPos := 5.0 + (target.Speed * 0.07)
	if updated.TargetPosX < expectedPos-0.1 || updated.TargetPosX > expectedPos+0.1 {
		t.Errorf("TargetPosX = %f, want ~%f", updated.TargetPosX, expectedPos)
	}
}

func TestModel_Update_AnimTick_TargetReachesEdge(t *testing.T) {
	baseTime := time.Now()
	m := Model{
		Pet:            pet.Pet{},
		Target:         Targets["butterfly"],
		TermWidth:      80,
		TermHeight:     24,
		LastUpdateTime: baseTime,
		TargetPosX:     77.0, // Near edge (maxX is 78)
		TargetPosY:     12,
	}

	// Tick should move target past edge and trigger quit
	// Butterfly moves 8.0 * 0.07 = 0.56 columns, so 77 + 0.56 > 78 (edge)
	nextTime := baseTime.Add(70 * time.Millisecond)
	updatedModel, cmd := m.Update(animTickMsg(nextTime))

	if cmd == nil {
		t.Error("Target reaching edge should return quit command")
	}

	// Model should still be returned even when quitting
	if updatedModel == nil {
		t.Error("Update should return model even when quitting")
	}
}

func TestModel_Update_AnimTick_PetMovement(t *testing.T) {
	baseTime := time.Now()
	m := Model{
		Pet:            pet.Pet{},
		Target:         Targets["butterfly"],
		TermWidth:      80,
		TermHeight:     24,
		LastUpdateTime: baseTime,
		PetPosX:        0,
		PetPosY:        12,
		TargetPosX:     20,
		TargetPosY:     12,
	}

	nextTime := baseTime.Add(70 * time.Millisecond)
	updatedModel, _ := m.Update(animTickMsg(nextTime))
	updated := updatedModel.(Model)

	// Pet should move towards target (distance > 3, so it will move)
	// Pet speed is 10.0 columns/sec, in 0.07 sec = 0.7 columns
	if updated.PetPosX <= 0 {
		t.Error("Pet should move horizontally towards target")
	}
}

func TestModel_Update_AnimTick_PetVerticalMovement(t *testing.T) {
	tests := []struct {
		name       string
		petPosX    float64 // Pet X position
		petPosY    float64
		targetPosX float64 // Target X determines its Y via sine wave
		wantChange string  // "up", "down", or "none"
	}{
		{
			name:       "Pet moves down when target is below",
			petPosX:    0,          // Pet at left
			petPosY:    3,          // Pet high up
			targetPosX: 100,        // Target far right
			wantChange: "down",     // Target will be at center ~12, pet moves down
		},
		{
			name:       "Pet moves up when target is above",
			petPosX:    0,          // Pet at left
			petPosY:    18,         // Pet low down
			targetPosX: 100,        // Target far right at center ~12
			wantChange: "up",       // Pet moves up toward center
		},
		{
			name:       "Pet doesn't move when close vertically",
			petPosX:    5,          // Position pet very close to target
			petPosY:    11,         // At center Y
			targetPosX: 5,          // Target at X=5 (early in sine wave, near center)
			wantChange: "none",     // At X~5, sin(1) ≈ 0.84, target at ~11+2.6=13.6, but distance check should work
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseTime := time.Now()

			m := Model{
				Pet:            pet.Pet{},
				Target:         Targets["butterfly"],
				TermWidth:      200, // Wide enough
				TermHeight:     24,
				LastUpdateTime: baseTime,
				PetPosX:        tt.petPosX,
				PetPosY:        tt.petPosY,
				TargetPosX:     tt.targetPosX,
			}

			// Calculate where target Y will be after sine wave
			height := float64(m.visibleRows())
			amplitude := height / 3.0
			centerY := height / 2.0
			frequency := 0.2
			targetY := centerY + amplitude*math.Sin((tt.targetPosX+m.Target.Speed*0.07)*frequency)

			nextTime := baseTime.Add(70 * time.Millisecond)
			updatedModel, _ := m.Update(animTickMsg(nextTime))
			updated := updatedModel.(Model)

			distY := math.Abs(targetY - tt.petPosY)

			switch tt.wantChange {
			case "down":
				if distY > 1 && updated.PetPosY <= tt.petPosY {
					t.Errorf("Pet should move down from Y=%f, got Y=%f (target at ~%f)", tt.petPosY, updated.PetPosY, targetY)
				}
			case "up":
				if distY > 1 && updated.PetPosY >= tt.petPosY {
					t.Errorf("Pet should move up from Y=%f, got Y=%f (target at ~%f)", tt.petPosY, updated.PetPosY, targetY)
				}
			case "none":
				// If distY was > 1, pet should have moved; if <= 1, should not have moved
				actuallyMoved := updated.PetPosY != tt.petPosY
				shouldMove := distY > 1
				if actuallyMoved != shouldMove {
					t.Errorf("Pet movement = %v, expected %v (distY=%f, pet Y: %f → %f)",
						actuallyMoved, shouldMove, distY, tt.petPosY, updated.PetPosY)
				}
			}
		})
	}
}

func TestModel_Update_AnimTick_CatchEndsRun(t *testing.T) {
	baseTime := time.Now()
	m := Model{
		Pet:            pet.Pet{},
		Target:         Targets["butterfly"],
		TermWidth:      40,
		TermHeight:     10,
		LastUpdateTime: baseTime,
		PetPosX:        5,
		PetPosY:        3,
		TargetPosX:     6,
		TargetPosY:     3,
	}

	nextTime := baseTime.Add(70 * time.Millisecond)
	_, cmd := m.Update(animTickMsg(nextTime))
	if cmd == nil {
		t.Fatalf("expected quit command when pet catches target")
	}
}

func TestModel_Update_AnimTick_BoundaryConstraints(t *testing.T) {
	// Test that target stays within boundaries during sine wave movement
	t.Run("Target stays within vertical boundaries", func(t *testing.T) {
		baseTime := time.Now()
		m := Model{
			Pet:            pet.Pet{},
			Target:         Targets["butterfly"],
			TermWidth:      80,
			TermHeight:     24,
			LastUpdateTime: baseTime,
			TargetPosX:     5,
			TargetPosY:     12,
			PetPosX:        0,
			PetPosY:        12,
		}

		minY := 0.0
		maxY := float64(m.visibleRows() - 1)

		// Run many ticks to traverse the full sine wave
		currentTime := baseTime
		for i := 0; i < 50; i++ {
			currentTime = currentTime.Add(70 * time.Millisecond)
			model, _ := m.Update(animTickMsg(currentTime))
			m = model.(Model)

			if m.TargetPosY < minY {
				t.Errorf("Tick %d: TargetPosY = %f, should be >= %f", i, m.TargetPosY, minY)
			}
			if m.TargetPosY > maxY {
				t.Errorf("Tick %d: TargetPosY = %f, should be <= %f", i, m.TargetPosY, maxY)
			}
		}
	})

	// Test that pet stays within boundaries when following target
	t.Run("Pet stays within vertical boundaries", func(t *testing.T) {
		baseTime := time.Now()
		m := Model{
			Pet:            pet.Pet{},
			Target:         Targets["butterfly"],
			TermWidth:      80,
			TermHeight:     24,
			LastUpdateTime: baseTime,
			TargetPosX:     20,
			TargetPosY:     3, // Near upper boundary
			PetPosX:        0,
			PetPosY:        12,
		}

		minY := 0.0
		maxY := float64(m.visibleRows() - 1)

		// Run ticks until pet moves and gets clamped
		currentTime := baseTime
		for i := 0; i < 20; i++ {
			currentTime = currentTime.Add(70 * time.Millisecond)
			model, _ := m.Update(animTickMsg(currentTime))
			m = model.(Model)

			if m.PetPosY < minY {
				t.Errorf("Tick %d: PetPosY = %f, should be >= %f", i, m.PetPosY, minY)
			}
			if m.PetPosY > maxY {
				t.Errorf("Tick %d: PetPosY = %f, should be <= %f", i, m.PetPosY, maxY)
			}
		}
	})
}

func TestModel_View_Initialization(t *testing.T) {
	m := Model{
		Pet:        pet.Pet{},
		Target:     Targets["butterfly"],
		TermWidth:  0, // Uninitialized
		TermHeight: 0,
	}

	view := m.View()
	if !strings.Contains(view, "Initializing") {
		t.Error("View should show 'Initializing...' when dimensions are zero")
	}
}

func TestModel_View_ContainsPetAndTarget(t *testing.T) {
	m := Model{
		Pet:        pet.Pet{},
		Target:     Targets["butterfly"],
		TermWidth:  80,
		TermHeight: 24,
		PetPosX:    5,
		PetPosY:    10,
		TargetPosX: 15,
		TargetPosY: 10,
	}

	view := m.View()

	// View should contain the target emoji
	if !strings.Contains(view, m.Target.Emoji) {
		t.Errorf("View should contain target emoji %q", m.Target.Emoji)
	}

	// View should contain some pet emoji (check for common chase emojis)
	petEmojis := []string{
		pet.StatusEmojiHappy,
		pet.StatusEmojiNeutral,
		pet.StatusEmojiSleeping,
		pet.StatusEmojiEnergetic,
		pet.StatusEmojiSad,
		pet.StatusEmojiHungry,
		pet.StatusEmojiExcited,
	}
	found := false
	for _, emoji := range petEmojis {
		if strings.Contains(view, emoji) {
			found = true
			break
		}
	}
	if !found {
		t.Error("View should contain a pet emoji")
	}

	// View should contain exit instruction
	if !strings.Contains(view, "Press any key to exit") {
		t.Error("View should contain exit instruction")
	}
}

func TestModel_View_GridDimensions(t *testing.T) {
	m := Model{
		Pet:        pet.Pet{},
		Target:     Targets["butterfly"],
		TermWidth:  40,
		TermHeight: 20,
		PetPosX:    5,
		PetPosY:    5,
		TargetPosX: 10,
		TargetPosY: 5,
	}

	view := m.View()
	lines := strings.Split(view, "\n")

	// View renders rows-1 grid lines, then a blank, then the instruction line.
	expectedLines := m.visibleRows() + 1
	if len(lines) != expectedLines {
		t.Errorf("View has %d lines, want %d", len(lines), expectedLines)
	}
}

func TestVisibleRowsMinimum(t *testing.T) {
	m := Model{TermHeight: 3}
	if got := m.visibleRows(); got != 6 {
		t.Fatalf("visibleRows min should be 6, got %d", got)
	}
}

func TestClampOnResize(t *testing.T) {
	m := Model{
		TermWidth:  10,
		TermHeight: 10,
		PetPosY:    20,
		TargetPosY: -5,
	}

	m.clampPositions()
	expectedMaxY := float64(m.visibleRows() - 1)
	if m.PetPosY != expectedMaxY {
		t.Fatalf("pet Y should clamp to %f, got %f", expectedMaxY, m.PetPosY)
	}
	if m.TargetPosY != 0 {
		t.Fatalf("target Y should clamp to 0, got %f", m.TargetPosY)
	}
}

func TestModel_View_OutOfBoundsPositions(t *testing.T) {
	m := Model{
		Pet:        pet.Pet{},
		Target:     Targets["butterfly"],
		TermWidth:  80,
		TermHeight: 24,
		PetPosX:    -5,  // Out of bounds
		PetPosY:    100, // Out of bounds
		TargetPosX: 200, // Out of bounds
		TargetPosY: -10, // Out of bounds
	}

	// Should not panic with out of bounds positions
	view := m.View()
	if view == "" {
		t.Error("View should still render with out of bounds positions")
	}
}

func TestModel_PetHorizontalMovementThreshold(t *testing.T) {
	// Pet only moves horizontally if distance > 3
	// NOTE: Target moves first (8.0 * 0.07 = 0.56 columns per tick),
	// then pet evaluates distance to NEW target position
	tests := []struct {
		name     string
		distX    float64
		wantMove bool
	}{
		{
			name:     "Pet moves when distX > 3",
			distX:    5,   // After target moves +0.56, still > 3
			wantMove: true,
		},
		{
			name:     "Pet doesn't move when distX = 3",
			distX:    2.4, // After target moves +0.56 → 2.96, still < 3, so no movement
			wantMove: false,
		},
		{
			name:     "Pet doesn't move when distX < 3",
			distX:    2,
			wantMove: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseTime := time.Now()
			m := Model{
				Pet:            pet.Pet{},
				Target:         Targets["butterfly"],
				TermWidth:      80,
				TermHeight:     24,
				LastUpdateTime: baseTime,
				PetPosX:        10,
				PetPosY:        12,
				TargetPosX:     10 + tt.distX,
				TargetPosY:     12,
			}

			nextTime := baseTime.Add(70 * time.Millisecond)
			updatedModel, _ := m.Update(animTickMsg(nextTime))
			updated := updatedModel.(Model)

			moved := updated.PetPosX > m.PetPosX
			if moved != tt.wantMove {
				// Calculate actual distance after target moved
				targetMoved := m.Target.Speed * 0.07
				actualDist := tt.distX + targetMoved
				t.Errorf("Pet moved = %v, want %v (initial distX = %f, after target moved = %f)",
					moved, tt.wantMove, tt.distX, actualDist)
			}
		})
	}
}

func TestGetChaseEmoji(t *testing.T) {
	tests := []struct {
		name     string
		pet      pet.Pet
		distX    int
		distY    int
		expected string
	}{
		{
			name:     "About to catch - close distance",
			pet:      pet.Pet{Energy: 50, Happiness: 50, Hunger: 50},
			distX:    1,
			distY:    0,
			expected: pet.StatusEmojiExcited,
		},
		{
			name:     "Close but not touching still excites",
			pet:      pet.Pet{Energy: 50, Happiness: 50, Hunger: 50},
			distX:    3,
			distY:    1,
			expected: pet.StatusEmojiExcited,
		},
		{
			name:     "Tired pet - low energy",
			pet:      pet.Pet{Energy: 20, Happiness: 50, Hunger: 50},
			distX:    10,
			distY:    5,
			expected: pet.StatusEmojiSleeping,
		},
		{
			name:     "Energetic pet - high energy",
			pet:      pet.Pet{Energy: 90, Happiness: 50, Hunger: 50},
			distX:    10,
			distY:    5,
			expected: pet.StatusEmojiEnergetic,
		},
		{
			name:     "Sad pet - low happiness",
			pet:      pet.Pet{Energy: 50, Happiness: 20, Hunger: 50},
			distX:    10,
			distY:    5,
			expected: pet.StatusEmojiSad,
		},
		{
			name:     "Happy pet - high happiness",
			pet:      pet.Pet{Energy: 50, Happiness: 90, Hunger: 50},
			distX:    10,
			distY:    5,
			expected: pet.StatusEmojiHappy,
		},
		{
			name:     "Hungry pet - low hunger",
			pet:      pet.Pet{Energy: 50, Happiness: 50, Hunger: 20},
			distX:    10,
			distY:    5,
			expected: pet.StatusEmojiHungry,
		},
		{
			name:     "Hungry takes priority over energetic",
			pet:      pet.Pet{Energy: 90, Happiness: 90, Hunger: 20},
			distX:    10,
			distY:    5,
			expected: pet.StatusEmojiHungry,
		},
		{
			name:     "Default neutral pet",
			pet:      pet.Pet{Energy: 50, Happiness: 50, Hunger: 50},
			distX:    10,
			distY:    5,
			expected: pet.StatusEmojiNeutral,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getChaseEmoji(tt.pet, tt.distX, tt.distY)
			if result != tt.expected {
				t.Errorf("getChaseEmoji() = %v, want %v", result, tt.expected)
			}
		})
	}
}
