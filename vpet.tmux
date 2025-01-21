#!/usr/bin/env bash

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

vpet_interpolation="#($CURRENT_DIR/scripts/pet_status.sh)"
status_right=$(tmux show-option -gqv "status-right")
tmux set-option -g status-right "$vpet_interpolation $status_right"
tmux set-option -g status-interval 60
