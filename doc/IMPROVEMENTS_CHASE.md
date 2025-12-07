# Chase Mode – Engagement Ideas

## Visual Dynamism
- **Vertical playfield:** Expand to multiple rows; let targets flutter/bounce on both axes; pet follows with slight lag to feel lifelike.
- **Parallax lanes:** Simulate depth with two height bands (foreground/background) and different speeds; occasional lane swaps for surprise.
- **Particle bits:** Minimal sparkles/dust trails on sprinting or sudden turns (cheap ASCII bursts).
- **Contextual emojis:** Swap pet icon by mood/energy (😴 slows, 😼 speeds up, 😻 on near catch) to show state without HUD.
- **Lighting cycles:** Slow hue/ANSI color shifts (sunset → night) every ~30s to add ambient variation; avoid flashing.

## Target Behaviors
- **Pattern variety:** Cycle between sine flutter, zigzag sprints, hover-pause, and “fake-out” micro-reversals.
- **Risk/reward pickups:** Occasional floating items (⭐ bond boost, 🍗 energy) that appear briefly; chasing them may delay catch, introducing choice.
- **Adaptive difficulty:** Increase target agility if user has long streaks; ease off after near misses to keep flow.
- **Boss target:** Rare golden butterfly/mouse with longer route and unique animation on catch.

## Player Agency
- **Action key bursts:** Allow short dash on space key with cooldown; costs energy, boosts catch odds.
- **Form swap:** If pet has forms, temporarily switch sprite after a successful catch (“hunter form”) for next round.
- **Cheer input:** Key to cheer/whistle nudges target slightly or freezes for a frame, gated by cooldown.

## Feedback & Scoring
- **Combo meter:** Track consecutive catches; higher combo increases speed lines/particles and yields bond/XP bonus when exiting.
- **Streak messaging:** Short toasts (“Hot pursuit! x3”, “Close shave!”) using existing status renderer.
- **Run summary:** On exit, show time survived, catches, missed pickups, longest streak.

## Session Flow
- **Rounds:** After each catch, brief victory pose + optional reward choice (energy vs. happiness) before next spawn.
- **Difficulty ramp:** Gradually shorten frame interval or widen vertical range per round; cap to keep accessible.
- **Pause-safe exit:** Any key still exits, but add “[p] pause / [q] quit” hints to reduce accidental quits.

## Accessibility & Comfort
- **Speed scaling:** Read terminal height to adjust speed (smaller terminals → slower) and provide `--chase-speed` flag.
- **Color fallback:** Offer monochrome mode for limited terminals; keep high-contrast glyph choices.
- **Motion reduction:** `--chase-low-motion` reduces zigzags and particle spam.

## Technical Notes
- Keep ASCII width <= 2 columns per glyph where possible; align with wcwidth to avoid jitter.
- Use a single ticker (tea.Tick) with delta-based movement instead of per-entity ticks to simplify pacing.
- Keep RNG seedable (`--chase-seed`) for reproducible demos and tests.
