# Vertical Playfield Plan

Goal: Expand chase mode to a multi-row playfield with targets moving on both X/Y axes and the pet following with slight lag to feel lifelike.

Key decisions
- **Coordinate system:** Track integer `y` positions (0..height-1) alongside existing `x`. Maintain `height` from terminal size; clamp target/pet positions.
- **Movement model:** Target follows sine/random-walk in Y with speed capped to 1 row per frame. Pet chases with lower Y acceleration to create lag.
- **Collision / catch:** Catch triggers when `x` overlap AND `|y_pet - y_target| <= 0` (same row) to keep rules clear.
- **Rendering:** Use slice of strings, each row with target/pet if present; background filler for empty rows. Keep glyph widths ≤2 columns.
- **Tick rate:** Keep existing chase tick; update positions each tick based on velocity/direction.
- **Config knobs:** `--chase-rows` optional override; default derived from terminal rows minus margin; minimum 3 rows.

Risks / mitigations
- **Terminal flicker:** Use single lipgloss renderer with JoinVertical; avoid per-row lipgloss styles to keep perf.
- **Small terminals:** Auto-reduce rows and clamp movement to avoid out-of-bounds.
- **Readability:** Keep characters aligned; avoid combining target and pet on same row when both present by left/right spacing.

Next steps
1) Add `y`, `vy` to chase model structs and derive `maxRows`.
2) Implement target Y movement generator (sine + noise).
3) Update pet AI to chase Y with lag and optional easing.
4) Update view to render multiple rows and adjust catch logic/tests.
