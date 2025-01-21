#!/usr/bin/env bash

# Get the current directory of the script
CURRENT_DIR=$HOME/.tmux

get_pet_status() {
  PET_FILE="$HOME/.config/tamagotchi/pet.json"
  # if [ ! -f "$PET_FILE" ]; then
  #   echo "😺"
  #   return
  # fi

  # Read the JSON file and extract status information
  SLEEPING=$(jq -r '.sleeping' "$PET_FILE")
  HUNGER=$(jq -r '.hunger' "$PET_FILE")
  ENERGY=$(jq -r '.energy' "$PET_FILE")
  HAPPINESS=$(jq -r '.happiness' "$PET_FILE")

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

update_status() {
  while true; do
    STATUS=$(get_pet_status)
    CURRENT_STATUS_RIGHT=$(tmux show-option -gqv "status-right")
    tmux set-option -g status-right "#[fg=magenta]#{?client_prefix,#[reverse]prefix#[noreverse],}#[default] $STATUS $CURRENT_STATUS_RIGHT"
    sleep 60
  done
}

# Start the update process in the background
update_status &

# Save the PID to a file so we can kill it later if needed
echo $! > "$HOME/.config/tamagotchi/tmux_status.pid"

# Update status every minute
tmux set-option -g status-interval 60
