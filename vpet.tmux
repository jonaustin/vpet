#!/usr/bin/env bash

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Get current status-right value
current_status=$(tmux show-option -gqv "status-right")

# Add our pet status to the beginning of status-right
if [[ -z "$current_status" ]]; then
    tmux set-option -g status-right "#($CURRENT_DIR/scripts/pet_status.sh)"
else
    tmux set-option -g status-right "#($CURRENT_DIR/scripts/pet_status.sh) $current_status"
fi

# Update every minute
tmux set-option -g status-interval 60
