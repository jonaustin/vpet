#!/usr/bin/env bash

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Kill any existing update processes
if [ -f "$HOME/.config/vpet/tmux_update.pid" ]; then
    pkill -F "$HOME/.config/vpet/tmux_update.pid" 2>/dev/null
    rm "$HOME/.config/vpet/tmux_update.pid"
fi

# Get current status-right value
current_status=$(tmux show-option -gqv "status-right")

# Start background process to update status
(
    while true; do
        status=$("$CURRENT_DIR/scripts/pet_status.sh")
        if [[ -z "$current_status" ]]; then
            tmux set-option -g status-right "$status"
        else
            tmux set-option -g status-right "$status $current_status"
        fi
        sleep 60
    done
) &

# Save the background process PID
echo $! > "$HOME/.config/vpet/tmux_update.pid"

# Set initial update interval
tmux set-option -g status-interval 60
