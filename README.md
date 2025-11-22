# vpet - Terminal Virtual Pet

A Tamagotchi-style virtual pet that lives in your terminal with tmux integration.

## Features

**Lifecycle Mechanics**
- 3 Life Stages: Baby (0-48h), Child (48-96h), Adult (96h+)
- Evolution system with 10 different forms based on care quality
- Natural aging with eventual death from old age (~1 week)
- 4 Death Causes: Neglect, Starvation, Sickness, Old Age
- Random illnesses requiring medicine

**Core Stats System**
- **Health**: Combined metric affected by all stats
- **Hunger**: Drains faster when awake (5%/hr vs 3%/hr sleeping)
- **Happiness**: Affected by other stats and interactions
- **Energy**: Recovers while sleeping, drains when playing
- **Age**: Tracks lifespan in hours

**Autonomous Behavior**
- Auto-sleep when energy drops to 20% or below
- Auto-wake after 6-8 hours of sleep when energy restored
- Dynamic mood system: Normal, Playful, Lazy, Needy
- Action refusal (won't eat when full, won't play when exhausted)

**Life Events**
- Random events occur based on pet's state and mood
- 9 event types: Chasing butterflies, Found something, Scared, Daydreaming, Ate something, Singing, Nightmare, Zoomies, Wants cuddles
- Respond to events for rewards, or ignore them (with consequences)

**Care System**
- Feed (+30% Hunger) - Refused when hunger >90%
- Play (+30% Happiness) - Refused when energy <20% or lazy mood
- Sleep (Energy recovery)
- Medicine (Cure sickness +30% Health)

## Evolution System

Your pet evolves based on care quality at each life stage:

```
Baby (0-48h)
    │
    ├─ Good Care (70%+) ───► Healthy Child
    │                            ├─ Perfect (85%+) ─► Elite Adult ⭐
    │                            ├─ Good (70%+) ────► Standard Adult 😺
    │                            └─ Poor (<70%) ────► Grumpy Adult 😼
    │
    ├─ Poor Care (40-69%) ─► Troubled Child
    │                            ├─ Improved (70%+) ► Redeemed Adult 😸
    │                            └─ Continued (<70%) ► Delinquent Adult 😾
    │
    └─ Neglect (<40%) ─────► Sickly Child ─────────► Weak Adult 🤕
```

## Tmux Integration

### Status Display
The tmux status shows two icons: **Activity** + **Feeling**

**Activity Icons (what pet is doing):**
| Icon | Meaning |
|------|---------|
| 😸 | Awake (default) |
| 😴 | Sleeping |
| 🦋 | Chasing butterfly |
| 🎁 | Found something |
| ⚡ | Scared |
| 💭 | Daydreaming |
| 🎵 | Singing |
| 😰 | Nightmare |
| 💨 | Zoomies |
| 🥺 | Wants cuddles |
| 🤢 | Ate something weird |

**Feeling Icons (critical needs):**
| Icon | Meaning |
|------|---------|
| 🙀 | Hungry (<30%) |
| 😾 | Tired (<30%) |
| 😿 | Sad (<30%) |
| 🤢 | Sick (<30%) |
| 🥱 | Drowsy (30-40% energy) |
| (none) | All is well |
| 💀 | Dead |

**Examples:** `😴🙀` = Sleeping but hungry, `🦋` = Chasing butterfly (all good), `😸🥱` = Awake but drowsy

### Clickable Status
Click the pet icon in tmux status bar to toggle the stats popup window.

### Setup
Update `VPET_DIR` in `vpet.tmux` and run:
```bash
bash vpet.tmux
```

**Hotkey:** `Prefix + P` to view detailed stats popup

## Installation

```bash
# Clone repo
git clone https://github.com/yourusername/vpet.git
cd vpet

# Build binary
go build -o vpet main.go
```

## Usage

```bash
# Start interactive mode
vpet

# Update stats without UI (for tmux)
vpet -u

# Check current status
vpet -status

# Display detailed stats
vpet -stats
```

## Controls

```
↑/↓ or k/j   Navigate menu
Enter/Space  Select action
e/r          Respond to events
q            Quit and save
```

### Hidden Debug Menu
Press `c` to access the cheat menu (for testing/debugging):
- Max/Min all stats
- Set mood
- Toggle illness/sleep
- Cycle chronotype
- Advance age
- Kill pet

## Moods

Pets have dynamic moods that affect behavior:

| Mood | Effect |
|------|--------|
| Normal | Standard behavior |
| Playful | Extra happiness from play, triggers zoomies |
| Lazy | May refuse to play when energy <50% |
| Needy | Triggers cuddles event, wants attention |

Moods change every 2-4 hours based on stats:
- Tired pets → more likely to be lazy
- Unhappy pets → more likely to be needy
- Happy/rested pets → random mood

## Chronotypes

Each pet is born with a random **chronotype** that determines their natural sleep/wake cycle:

| Chronotype | Emoji | Active Hours | Description |
|------------|-------|--------------|-------------|
| Early Bird | 🌅 | 5am - 9pm | Morning pet, sleeps early |
| Normal | ☀️ | 7am - 11pm | Standard schedule |
| Night Owl | 🦉 | 10am - 2am | Late riser, stays up late |

**Effects:**

**During Active Hours (preferred wake time):**
- Normal energy decay
- Full happiness bonus from play
- Won't auto-sleep unless energy critically low

**Outside Active Hours (should be sleeping):**
- 50% faster energy drain when awake
- 30% reduced happiness from play
- Auto-sleeps at higher energy threshold (40% vs 20%)
- Won't auto-wake even if energy restored

**During Preferred Sleep Hours:**
- 20% faster energy recovery while sleeping

## Stat Decay Rates

| Stat      | Awake Rate | Sleeping Rate | Care Action   |
|-----------|------------|---------------|---------------|
| Health    | -2%/hr*    | -1%/hr*       | Medicine +30% |
| Hunger    | -5%/hr     | -3%/hr        | Feed +30%     |
| Energy    | -5%/2hrs   | +10%/hr       | Sleep         |
| Happiness | -2%/hr**   | -2%/hr**      | Play +30%     |

*Only when any stat <15%
**Only when Hunger/Energy <30%

## Persistent State

Your pet continues aging even when closed! Stats save to:
`~/.config/vpet/pet.json`
