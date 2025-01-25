# vpet - Terminal Virtual Pet

A Tamagotchi-style virtual pet that lives in your terminal with tmux integration!

## Features

🖤 **Lifecycle Mechanics**  
- 3 Life Stages: Baby (0-24h), Child (24-48h), Adult (48h+)
- Natural aging with eventual death from old age (~3 days)
- 4 Death Causes: Neglect, Starvation, Sickness, Old Age
- Random illnesses requiring medicine
- Permanent death state with memorial

📊 **Core Stats System**  
- **Health**: Combined metric affected by all stats
- **Hunger**: Drains faster when awake (5%/hr vs 3%/hr sleeping)
- **Happiness**: Affected by other stats and interactions  
- **Energy**: Recovers while sleeping, drains when playing
- **Age**: Tracks lifespan in hours

💊 **Care System**  
- Feed (🍗 +30% Hunger)
- Medicine (💊 Cure sickness +30% Health) 
- Play (🎾 +30% Happiness)
- Discipline (🪄 Prevent bad behavior)
- Sleep (😴 Energy recovery)

## Tmux Integration

![Screenshot](tmux-pet-example.png)

1. Save your pet's status in tmux status line:
```bash
# ~/.tmux.conf
set -g status-right "#(path/to/scripts/pet_status.sh)"
```

2. Create `pet_status.sh`:
```bash
#!/bin/bash
VPET_BIN=path/to/vpet
$VPET_BIN -u  # Update stats first
$VPET_BIN -status  # Show current status
```

3. Make executable:
```bash
chmod +x ~/scripts/pet_status.sh
```

## Installation

```bash
# 1. Clone repo
git clone https://github.com/yourusername/vpet.git
cd vpet

# 2. Build binary
go build -o vpet main.go

# 3. Install to PATH
sudo mv vpet /usr/local/bin/
```

## Gameplay

```bash
# Start interactive mode
vpet

# Update stats without UI (for tmux)
vpet -u

# Check current status
vpet -status
```

## Stat System

| Stat      | Awake Rate | Sleeping Rate | Care Action   |
|-----------|------------|---------------|---------------|
| Health    | -5%/hr*    | -3%/hr*       | Medicine +30% |
| Hunger    | -5%/hr     | -3%/hr        | Feed +30%     |
| Energy    | -5%/2hrs   | +10%/hr       | Sleep         |
| Happiness | -2%/hr**   | -2%/hr**      | Play +30%     |

*When any stat <30%  
**When Hunger/Energy <30%

## Controls

```
↑/↓    Navigate menu  
Enter  Select action  
q      Quit and save
```

⚠️ **Persistent State**  
Your pet continues aging even when closed! Stats save to:  
`~/.config/vpet/pet.json`
