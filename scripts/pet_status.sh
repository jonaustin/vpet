#!/usr/bin/env bash

PET_FILE="$HOME/.config/vpet/pet.json"
if [ ! -f "$PET_FILE" ]; then
    echo "😺"
    exit 0
fi

if ! command -v jq &> /dev/null; then
    echo "😺"
    exit 0
fi

# Read the JSON file and extract status information
if ! SLEEPING=$(jq -r '.sleeping' "$PET_FILE" 2>/dev/null) || \
   ! HUNGER=$(jq -r '.hunger' "$PET_FILE" 2>/dev/null) || \
   ! ENERGY=$(jq -r '.energy' "$PET_FILE" 2>/dev/null) || \
   ! HAPPINESS=$(jq -r '.happiness' "$PET_FILE" 2>/dev/null); then
    echo "😺"
    exit 0
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
