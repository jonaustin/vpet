#!/usr/bin/env bash

VPET_DIR=~/code/vpet
CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ORIGINAL_STATUS_FILE="$HOME/.config/vpet/tmux_original_status"
PID_FILE="$HOME/.config/vpet/tmux_update.pid"

# Ensure config directory exists first
mkdir -p "$HOME/.config/vpet"

# Cleanup function to restore original status
cleanup() {
    # Restore original tmux status-left
    if [ -f "$ORIGINAL_STATUS_FILE" ]; then
        ORIGINAL_STATUS=$(cat "$ORIGINAL_STATUS_FILE" 2>/dev/null)
        tmux set-option -g status-left "$ORIGINAL_STATUS"
    fi
    # Clean up PID file
    rm -f "$PID_FILE"
}

# Store the original status-left (only on first run)
if [ ! -f "$ORIGINAL_STATUS_FILE" ]; then
    tmux show-option -gqv "status-left" > "$ORIGINAL_STATUS_FILE"
fi

# Read the saved original status
ORIGINAL_STATUS=$(cat "$ORIGINAL_STATUS_FILE" 2>/dev/null || echo "")

# Function to update status
update_status() {
    go run ${VPET_DIR}/main.go -u
    local pet_status=$(go run ${VPET_DIR}/main.go -status)
    if [[ -z "$ORIGINAL_STATUS" ]]; then
        tmux set-option -g status-left "$pet_status"
    else
        tmux set-option -g status-left "$pet_status $ORIGINAL_STATUS"
    fi
}

# Kill any existing update processes
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE" 2>/dev/null)
    if [ -n "$OLD_PID" ]; then
        # Check if process exists
        if kill -0 "$OLD_PID" 2>/dev/null; then
            # Verify it's actually a vpet process
            if ps -p "$OLD_PID" -o command= 2>/dev/null | grep -q "vpet"; then
                # Send TERM signal
                kill "$OLD_PID" 2>/dev/null
                # Wait for process to terminate (max 2 seconds)
                for i in {1..20}; do
                    kill -0 "$OLD_PID" 2>/dev/null || break
                    sleep 0.1
                done
                # Force kill if still alive
                if kill -0 "$OLD_PID" 2>/dev/null; then
                    kill -9 "$OLD_PID" 2>/dev/null
                fi
            fi
        fi
    fi
    # Clean up PID file
    rm -f "$PID_FILE"
fi

# Set initial status
update_status

# Start background process to update status
(
    # Set trap to cleanup on exit in the background process
    trap cleanup EXIT
    trap 'exit' INT TERM

    while true; do
        update_status
        sleep 3
    done
) &

# Save the background process PID
echo $! > "$PID_FILE"

# Ensure status updates frequently
tmux set-option -g status-interval 5

# Bind prefix + P to show stats popup
tmux bind-key P display-popup -E -w 60% -h 60% -xC -yC \
    "cd ${VPET_DIR} && go run main.go -stats; echo '\nPress any key to close'; read -n 1"
