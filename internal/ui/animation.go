package ui

import "time"

// AnimationType represents the type of action animation
type AnimationType int

const (
	AnimNone AnimationType = iota
	AnimFeed
	AnimPlay
	AnimSleep
	AnimMedicine
)

// Animation holds the current animation state
type Animation struct {
	Type      AnimationType
	Frame     int
	StartTime time.Time
}

// AnimationFrames contains ASCII art frames for each animation type
var AnimationFrames = map[AnimationType][]string{
	AnimFeed: {
		`
   🍖
     \
      😺
`,
		`

   🍖→😺

`,
		`

     😸
   *nom*
`,
		`

     😋
   *munch*
`,
	},
	AnimPlay: {
		`
  🎾        😺
`,
		`
     🎾     😸
`,
		`
        🎾  😺
`,
		`
     🎾     😸
              *boing*
`,
		`
  🎾        😺
              *catch!*
`,
	},
	AnimSleep: {
		`
     😺
`,
		`
     😪
      z
`,
		`
     😴
     z
      z
`,
		`
     😴
    z
     z
      z
`,
	},
	AnimMedicine: {
		`
  💊       😿
`,
		`
     💊    😿
`,
		`
       💊→ 😺
`,
		`
           😺
          +30
`,
		`
           😸
        ✨ +30 ✨
`,
	},
}

// AnimationDuration is how long each frame displays
const AnimationFrameDuration = 200 * time.Millisecond

// GetAnimationFrame returns the current frame for an animation
func GetAnimationFrame(anim Animation) string {
	frames := AnimationFrames[anim.Type]
	if len(frames) == 0 {
		return ""
	}
	if anim.Frame >= len(frames) {
		return frames[len(frames)-1]
	}
	return frames[anim.Frame]
}

// IsAnimationComplete returns true if the animation has finished
func IsAnimationComplete(anim Animation) bool {
	frames := AnimationFrames[anim.Type]
	return anim.Frame >= len(frames)
}

// AnimationTotalFrames returns the number of frames for an animation type
func AnimationTotalFrames(animType AnimationType) int {
	return len(AnimationFrames[animType])
}
