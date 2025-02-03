#!/usr/bin/env bash

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Store the original status-left
ORIGINAL_STATUS=$(tmux show-option -gqv "status-left")

# Function to update status
update_status() {
    go run ~/exp/vpet/main.go -u
    local pet_status=$(go run ~/exp/vpet/main.go -status)
    if [[ -z "$ORIGINAL_STATUS" ]]; then
        tmux set-option -g status-left "$pet_status"
    else
        tmux set-option -g status-left "$pet_status $ORIGINAL_STATUS"
    fi
}

# Kill any existing update processes
if [ -f "$HOME/.config/vpet/tmux_update.pid" ]; then
    pkill -F "$HOME/.config/vpet/tmux_update.pid" 2>/dev/null
    rm "$HOME/.config/vpet/tmux_update.pid"
fi

# Set initial status
update_status

# Start background process to update status
(
    while true; do
        update_status
        sleep 5
    done
) &

# Save the background process PID
echo $! > "$HOME/.config/vpet/tmux_update.pid"

# Ensure status updates frequently
tmux set-option -g status-interval 5
