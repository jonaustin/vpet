package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"vpet/internal/pet"
)

func TestAnimationTypes(t *testing.T) {
	tests := []struct {
		name     string
		animType AnimationType
		expected int // minimum expected frames
	}{
		{"Feed animation has frames", AnimFeed, 3},
		{"Play animation has frames", AnimPlay, 4},
		{"Sleep animation has frames", AnimSleep, 3},
		{"Medicine animation has frames", AnimMedicine, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frames := AnimationTotalFrames(tt.animType)
			if frames < tt.expected {
				t.Errorf("Expected at least %d frames for %v, got %d", tt.expected, tt.animType, frames)
			}
		})
	}
}

func TestGetAnimationFrame(t *testing.T) {
	anim := Animation{
		Type:      AnimFeed,
		Frame:     0,
		StartTime: time.Now(),
	}

	frame := GetAnimationFrame(anim)
	if frame == "" {
		t.Error("Expected non-empty frame for AnimFeed at frame 0")
	}

	// Test frame beyond total
	anim.Frame = 100
	frame = GetAnimationFrame(anim)
	if frame == "" {
		t.Error("Expected last frame for out-of-bounds frame index")
	}
}

func TestIsAnimationComplete(t *testing.T) {
	tests := []struct {
		name     string
		anim     Animation
		expected bool
	}{
		{
			name: "Animation at start is not complete",
			anim: Animation{
				Type:  AnimFeed,
				Frame: 0,
			},
			expected: false,
		},
		{
			name: "Animation at middle is not complete",
			anim: Animation{
				Type:  AnimFeed,
				Frame: 1,
			},
			expected: false,
		},
		{
			name: "Animation past end is complete",
			anim: Animation{
				Type:  AnimFeed,
				Frame: AnimationTotalFrames(AnimFeed),
			},
			expected: true,
		},
		{
			name: "No animation is complete",
			anim: Animation{
				Type:  AnimNone,
				Frame: 0,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAnimationComplete(tt.anim)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAnimationFrameDuration(t *testing.T) {
	// Ensure animation frame duration is reasonable (100-500ms)
	if AnimationFrameDuration < 100*time.Millisecond {
		t.Error("Animation frame duration too short")
	}
	if AnimationFrameDuration > 500*time.Millisecond {
		t.Error("Animation frame duration too long")
	}
}

func TestAllAnimationsHaveContent(t *testing.T) {
	animTypes := []AnimationType{AnimFeed, AnimPlay, AnimSleep, AnimMedicine}

	for _, animType := range animTypes {
		frames := AnimationFrames[animType]
		if len(frames) == 0 {
			t.Errorf("Animation type %v has no frames", animType)
			continue
		}

		for i, frame := range frames {
			if frame == "" {
				t.Errorf("Animation type %v has empty frame at index %d", animType, i)
			}
		}
	}
}

func TestAnimTickIgnoresStaleTicks(t *testing.T) {
	start := time.Now()
	m := Model{Animation: Animation{Type: AnimFeed, Frame: 0, StartTime: start}}

	// Begin a new animation before the stale tick arrives
	newStart := start.Add(time.Second)
	m.Animation = Animation{Type: AnimPlay, Frame: 0, StartTime: newStart}

	updated, cmd := m.Update(animTickMsg{started: start})
	updatedModel := updated.(Model)

	if updatedModel.Animation.Frame != 0 {
		t.Fatalf("stale tick advanced frame: got %d", updatedModel.Animation.Frame)
	}
	if cmd != nil {
		t.Fatalf("expected no follow-up tick for stale message, got %v", cmd)
	}
}

func TestAnimTickAdvancesCurrentAnimation(t *testing.T) {
	start := time.Now()
	m := Model{Animation: Animation{Type: AnimFeed, Frame: 0, StartTime: start}}

	updated, cmd := m.Update(animTickMsg{started: start})
	updatedModel := updated.(Model)

	if updatedModel.Animation.Frame != 1 {
		t.Fatalf("expected frame to advance to 1, got %d", updatedModel.Animation.Frame)
	}
	if cmd == nil {
		t.Fatalf("expected follow-up tick command for ongoing animation")
	}
}

func TestActionsIgnoredDuringAnimation(t *testing.T) {
	start := time.Now()
	m := Model{Animation: Animation{Type: AnimFeed, Frame: 0, StartTime: start}, Choice: 0}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updatedModel := updated.(Model)

	if updatedModel.Animation.StartTime != start {
		t.Fatalf("animation should not change while active; startTime changed")
	}
	if cmd != nil {
		t.Fatalf("expected no command when input ignored during animation, got %v", cmd)
	}
}

func TestRenderAnimationSkipsExpiredMessage(t *testing.T) {
	m := Model{
		Pet:            pet.Pet{Name: "Milo"},
		Animation:      Animation{Type: AnimFeed, Frame: 0, StartTime: time.Now()},
		Message:        "hello",
		MessageExpires: time.Now().Add(-time.Minute),
	}

	view := m.renderAnimation()
	if strings.Contains(view, "hello") {
		t.Fatalf("expected expired message to be omitted from animation view")
	}
}
