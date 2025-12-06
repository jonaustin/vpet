package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"vpet/internal/chase"
	"vpet/internal/pet"
	"vpet/internal/ui"
)

func main() {
	// Configure logging to write to config directory
	configDir := filepath.Dir(pet.GetConfigPath())
	logFile := filepath.Join(configDir, "vpet.log")
	logFileHandle, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer logFileHandle.Close()
	log.SetOutput(logFileHandle)

	updateOnly := flag.Bool("u", false, "Update pet stats only, don't run UI")
	statusFlag := flag.Bool("status", false, "Output current status emoji")
	statsFlag := flag.Bool("stats", false, "Display detailed pet statistics")
	chaseFlag := flag.Bool("chase", false, "Watch your pet chase a butterfly")
	flag.Parse()

	if *statsFlag {
		p := pet.LoadState()
		ui.DisplayStats(p)
		return
	}

	if *statusFlag {
		p := pet.LoadState()
		fmt.Print(strings.Split(pet.GetStatus(p), " ")[0])
		return
	}

	if *updateOnly {
		p := pet.LoadState()
		pet.SaveState(&p)
		return
	}

	if *chaseFlag {
		chase.Run()
		return
	}

	program := tea.NewProgram(ui.NewModel())
	if _, err := program.Run(); err != nil {
		log.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
