package ui

import (
	"testing"
	"time"
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
