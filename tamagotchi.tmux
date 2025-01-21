#!/usr/bin/env bash

#!/usr/bin/env bash

# Get the current directory of the script
CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

get_pet_status() {
  PET_FILE="$HOME/.config/tamagotchi/pet.json"
  if [ ! -f "$PET_FILE" ]; then
    echo "😺"
    return
  fi

  if ! command -v jq &> /dev/null; then
    echo "😺"
    return
  fi

  # Read the JSON file and extract status information
  if ! SLEEPING=$(jq -r '.sleeping' "$PET_FILE" 2>/dev/null) || \
     ! HUNGER=$(jq -r '.hunger' "$PET_FILE" 2>/dev/null) || \
     ! ENERGY=$(jq -r '.energy' "$PET_FILE" 2>/dev/null) || \
     ! HAPPINESS=$(jq -r '.happiness' "$PET_FILE" 2>/dev/null); then
    echo "😺"
    return
  fi

  # Determine status emoji based on pet state
  if [ "$SLEEPING" = "true" ]; then
    echo "😴"
  elif [ "$HUNGER" -lt 30 ]; then
    echo "🙀"
  elif [ "$ENERGY" -lt 30 ]; then
    echo "😾"
  elif [ "$HAPPINESS" -lt 30 ]; then
    echo "😿"
  else
    echo "😸"
  fi
}

# Kill any existing status update processes
if [ -f "$HOME/.config/tamagotchi/tmux_status.pid" ]; then
  pkill -F "$HOME/.config/tamagotchi/tmux_status.pid" 2>/dev/null
  rm "$HOME/.config/tamagotchi/tmux_status.pid"
fi

# Set initial status
STATUS=$(get_pet_status)
CURRENT_STATUS_RIGHT=$(tmux show-option -gqv "status-right")
tmux set-option -g status-right "#[fg=magenta]#{?client_prefix,#[reverse]prefix#[noreverse],}#[default] $STATUS $CURRENT_STATUS_RIGHT"

# Start the update process in the background
(while true; do
  STATUS=$(get_pet_status)
  tmux set-option -g status-right "#[fg=magenta]#{?client_prefix,#[reverse]prefix#[noreverse],}#[default] $STATUS $CURRENT_STATUS_RIGHT"
  sleep 1
done) &

# Save the PID to a file
echo $! > "$HOME/.config/tamagotchi/tmux_status.pid"

# Update status every minute
tmux set-option -g status-interval 60
