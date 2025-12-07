package chase

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"vpet/internal/pet"
)

func TestTargets(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		wantName string
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
				t.Errorf("Speed = %d, want > 0", target.Speed)
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
	m := Model{
		Pet:        pet.Pet{},
		Target:     Targets["butterfly"],
		TermWidth:  80,
		TermHeight: 24,
		Frame:      10,
	}

	// Any key should quit
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if updatedModel.(Model).Frame != 10 {
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

func TestModel_Update_AnimTick_FrameIncrement(t *testing.T) {
	m := Model{
		Pet:        pet.Pet{},
		Target:     Targets["butterfly"],
		TermWidth:  80,
		TermHeight: 24,
		Frame:      0,
		PetPosX:    0,
		PetPosY:    12,
		TargetPosX: 5,
		TargetPosY: 12,
	}

	updatedModel, _ := m.Update(animTickMsg{})
	updated := updatedModel.(Model)

	if updated.Frame != 1 {
		t.Errorf("Frame = %d, want 1", updated.Frame)
	}
}

func TestModel_Update_AnimTick_TargetMovement(t *testing.T) {
	target := Targets["butterfly"]
	m := Model{
		Pet:        pet.Pet{},
		Target:     target,
		TermWidth:  80,
		TermHeight: 24,
		Frame:      0,
		TargetPosX: 5,
		TargetPosY: 12,
	}

	// Run enough frames to trigger target movement
	// Butterfly speed is 3, so it moves every 3 frames
	for i := 0; i < target.Speed; i++ {
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.Update(animTickMsg{})
		m = model.(Model)
		if cmd == nil {
			t.Error("animTickMsg should return tick command")
		}
	}

	// After Speed frames, target should have moved horizontally
	if m.TargetPosX <= 5 {
		t.Errorf("TargetPosX = %d, expected > 5 after %d frames", m.TargetPosX, target.Speed)
	}
}

func TestModel_Update_AnimTick_TargetReachesEdge(t *testing.T) {
	m := Model{
		Pet:        pet.Pet{},
		Target:     Targets["butterfly"],
		TermWidth:  80,
		TermHeight: 24,
		Frame:      2, // Frame % 3 == 2, so next tick will move target
		TargetPosX: 79, // Near edge
		TargetPosY: 12,
	}

	// Next frame should move target past edge and trigger quit
	updatedModel, cmd := m.Update(animTickMsg{})

	if cmd == nil {
		t.Error("Target reaching edge should return quit command")
	}

	// Model should still be returned even when quitting
	if updatedModel == nil {
		t.Error("Update should return model even when quitting")
	}
}

func TestModel_Update_AnimTick_PetMovement(t *testing.T) {
	m := Model{
		Pet:        pet.Pet{},
		Target:     Targets["butterfly"],
		TermWidth:  80,
		TermHeight: 24,
		Frame:      1, // Next frame will be 2, pet moves every 2 frames
		PetPosX:    0,
		PetPosY:    12,
		TargetPosX: 20,
		TargetPosY: 12,
	}

	updatedModel, _ := m.Update(animTickMsg{})
	updated := updatedModel.(Model)

	// Pet should move towards target
	if updated.PetPosX <= 0 {
		t.Error("Pet should move horizontally towards target")
	}
}

func TestModel_Update_AnimTick_PetVerticalMovement(t *testing.T) {
	tests := []struct {
		name       string
		petPosY    int
		targetPosY int
		wantChange string // "up", "down", or "none"
	}{
		{
			name:       "Pet moves down when target is below",
			petPosY:    10,
			targetPosY: 15,
			wantChange: "down",
		},
		{
			name:       "Pet moves up when target is above",
			petPosY:    15,
			targetPosY: 10,
			wantChange: "up",
		},
		{
			name:       "Pet doesn't move when close vertically",
			petPosY:    12,
			targetPosY: 13,
			wantChange: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				Pet:        pet.Pet{},
				Target:     Targets["butterfly"],
				TermWidth:  80,
				TermHeight: 24,
				Frame:      1, // Next frame will be 2, pet moves every 2 frames
				PetPosX:    0,
				PetPosY:    tt.petPosY,
				TargetPosX: 20,
				TargetPosY: tt.targetPosY,
			}

			updatedModel, _ := m.Update(animTickMsg{})
			updated := updatedModel.(Model)

			switch tt.wantChange {
			case "down":
				if updated.PetPosY <= tt.petPosY {
					t.Errorf("Pet should move down from Y=%d, got Y=%d", tt.petPosY, updated.PetPosY)
				}
			case "up":
				if updated.PetPosY >= tt.petPosY {
					t.Errorf("Pet should move up from Y=%d, got Y=%d", tt.petPosY, updated.PetPosY)
				}
			case "none":
				if updated.PetPosY != tt.petPosY {
					t.Errorf("Pet should not move vertically from Y=%d, got Y=%d", tt.petPosY, updated.PetPosY)
				}
			}
		})
	}
}

func TestModel_Update_AnimTick_BoundaryConstraints(t *testing.T) {
	// Test that target stays within boundaries during sine wave movement
	t.Run("Target stays within vertical boundaries", func(t *testing.T) {
		m := Model{
			Pet:        pet.Pet{},
			Target:     Targets["butterfly"],
			TermWidth:  80,
			TermHeight: 24,
			Frame:      0,
			TargetPosX: 5,
			TargetPosY: 12,
			PetPosX:    0,
			PetPosY:    12,
		}

		minY := 2
		maxY := 19 // termHeight - 5

		// Run many frames to traverse the full sine wave
		for i := 0; i < 50; i++ {
			model, _ := m.Update(animTickMsg{})
			m = model.(Model)

			if m.TargetPosY < minY {
				t.Errorf("Frame %d: TargetPosY = %d, should be >= %d", i, m.TargetPosY, minY)
			}
			if m.TargetPosY > maxY {
				t.Errorf("Frame %d: TargetPosY = %d, should be <= %d", i, m.TargetPosY, maxY)
			}
		}
	})

	// Test that pet stays within boundaries when following target
	t.Run("Pet stays within vertical boundaries", func(t *testing.T) {
		m := Model{
			Pet:        pet.Pet{},
			Target:     Targets["butterfly"],
			TermWidth:  80,
			TermHeight: 24,
			Frame:      0,
			TargetPosX: 20,
			TargetPosY: 3, // Near upper boundary
			PetPosX:    0,
			PetPosY:    12,
		}

		minY := 2
		maxY := 19 // termHeight - 5

		// Run frames until pet moves and gets clamped
		for i := 0; i < 20; i++ {
			model, _ := m.Update(animTickMsg{})
			m = model.(Model)

			if m.PetPosY < minY {
				t.Errorf("Frame %d: PetPosY = %d, should be >= %d", i, m.PetPosY, minY)
			}
			if m.PetPosY > maxY {
				t.Errorf("Frame %d: PetPosY = %d, should be <= %d", i, m.PetPosY, maxY)
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

	// View should contain pet emoji
	if !strings.Contains(view, "😸") {
		t.Error("View should contain pet emoji 😸")
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

	// Should have termHeight - 3 grid lines + 1 blank + 1 instruction line
	// Total = termHeight - 1
	expectedLines := m.TermHeight - 1
	if len(lines) != expectedLines {
		t.Errorf("View has %d lines, want %d", len(lines), expectedLines)
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
	tests := []struct {
		name      string
		distX     int
		wantMove  bool
	}{
		{
			name:     "Pet moves when distX > 3",
			distX:    4,
			wantMove: true,
		},
		{
			name:     "Pet doesn't move when distX = 3",
			distX:    3,
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
			m := Model{
				Pet:        pet.Pet{},
				Target:     Targets["butterfly"],
				TermWidth:  80,
				TermHeight: 24,
				Frame:      1, // Next tick will be frame 2, pet moves
				PetPosX:    10,
				PetPosY:    12,
				TargetPosX: 10 + tt.distX,
				TargetPosY: 12,
			}

			updatedModel, _ := m.Update(animTickMsg{})
			updated := updatedModel.(Model)

			moved := updated.PetPosX > m.PetPosX
			if moved != tt.wantMove {
				t.Errorf("Pet moved = %v, want %v (distX = %d)", moved, tt.wantMove, tt.distX)
			}
		})
	}
}
