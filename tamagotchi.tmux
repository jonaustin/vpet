#!/usr/bin/env bash

# Get the current directory of the script
CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

get_pet_status() {
  PET_FILE="$HOME/.config/tamagotchi/pet.json"
  if [ ! -f "$PET_FILE" ]; then
    echo "😺"
    return
  }

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

# Add pet status to tmux status-right
CURRENT_STATUS_RIGHT=$(tmux show-option -gqv "status-right")
tmux set-option -g status-right "#[fg=magenta]#{?client_prefix,#[reverse]prefix#[noreverse],}#[default] $(get_pet_status) $CURRENT_STATUS_RIGHT"

# Update status every minute
tmux set-option -g status-interval 60
