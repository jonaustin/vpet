package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestFile(t *testing.T) func() {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "vpet-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Set the test config path
	testConfigPath = filepath.Join(tmpDir, "test-pet.json")

	// Return cleanup function
	return func() {
		testConfigPath = "" // Reset the test path
		// os.RemoveAll(tmpDir)
	}
}

func TestDeathConditions(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	// Create pet that has been critical for 13 hours (exceeds 12h threshold)
	currentTime := time.Now().UTC()
	criticalStart := currentTime.Add(-13 * time.Hour)
	testCfg := &TestConfig{
		InitialHunger:    lowStatThreshold - 1,
		InitialHappiness: lowStatThreshold - 1,
		InitialEnergy:    lowStatThreshold - 1,
		Health:           20, // Force critical health
		LastSavedTime:    criticalStart,
	}
	pet := newPet(testCfg)
	pet.CriticalStartTime = &criticalStart
	saveState(&pet)

	// Fix LastSaved time in file
	data, err := os.ReadFile(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	var savedPet Pet
	if err := json.Unmarshal(data, &savedPet); err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	savedPet.LastSaved = criticalStart
	data, err = json.MarshalIndent(savedPet, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal pet: %v", err)
	}
	if err := os.WriteFile(testConfigPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loadedPet := loadState()

	if !loadedPet.Dead {
		t.Error("Expected pet to be dead after 12+ hours in critical state")
	}
	// Death cause could be Neglect or Starvation depending on which stat hit 0 first
	if loadedPet.CauseOfDeath != "Neglect" && loadedPet.CauseOfDeath != "Starvation" {
		t.Errorf("Expected death cause 'Neglect' or 'Starvation', got '%s'", loadedPet.CauseOfDeath)
	}
}

func TestNewPet(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()
	pet := newPet(nil)

	if pet.Name != defaultPetName {
		t.Errorf("Expected pet name to be %s, got %s", defaultPetName, pet.Name)
	}

	// Check new fields
	if pet.Health != maxStat {
		t.Errorf("Expected initial health to be %d, got %d", maxStat, pet.Health)
	}
	if pet.Age != 0 {
		t.Errorf("Expected initial age to be 0, got %d", pet.Age)
	}
	if pet.LifeStage != 0 {
		t.Errorf("Expected initial life stage to be 0, got %d", pet.LifeStage)
	}
	if pet.Illness {
		t.Error("New pet should not be ill")
	}

	if pet.Health != maxStat {
		t.Errorf("Expected initial health to be %d, got %d", maxStat, pet.Health)
	}
	if pet.Age != 0 {
		t.Errorf("Expected initial age to be 0, got %d", pet.Age)
	}
	if pet.LifeStage != 0 {
		t.Errorf("Expected initial life stage to be 0, got %d", pet.LifeStage)
	}
	if pet.Illness {
		t.Error("New pet should not be ill")
	}

	if pet.Hunger != maxStat {
		t.Errorf("Expected initial hunger to be %d, got %d", maxStat, pet.Hunger)
	}

	if pet.Happiness != maxStat {
		t.Errorf("Expected initial happiness to be %d, got %d", maxStat, pet.Happiness)
	}

	if pet.Energy != maxStat {
		t.Errorf("Expected initial energy to be %d, got %d", maxStat, pet.Energy)
	}

	if pet.Sleeping {
		t.Error("Expected new pet to be awake")
	}
}

func TestPetStatUpdates(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	testCfg := &TestConfig{
		InitialHunger:    50,  // Start with lower hunger
		InitialHappiness: 50,  // Start with lower happiness
		InitialEnergy:    100, // Start with full energy
		Health:           80,  // Start with suboptimal health
		IsSleeping:       false,
		LastSavedTime:    time.Now(),
		Illness:          true, // Start with illness
	}
	m := initialModel(testCfg)

	// Test feeding
	originalHunger := m.pet.Hunger
	originalHappiness := m.pet.Happiness
	m.feed()

	if m.pet.Hunger <= originalHunger {
		t.Error("Feeding should increase hunger %")
	}
	if m.pet.Happiness < originalHappiness {
		t.Error("Feeding should not decrease happiness %")
	}

	// Test playing
	originalEnergy := m.pet.Energy
	originalHappiness = m.pet.Happiness
	m.play()

	if m.pet.Energy >= originalEnergy {
		t.Error("Playing should decrease energy %")
	}
	if m.pet.Happiness < originalHappiness {
		t.Error("Playing should not decrease happiness")
	}

	// Test sleeping
	m.toggleSleep()
	if !m.pet.Sleeping {
		t.Error("Pet should be sleeping after toggleSleep")
	}

	m.toggleSleep()
	if m.pet.Sleeping {
		t.Error("Pet should be awake after second toggleSleep")
	}
}

func TestStatBoundaries(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()
	m := initialModel(nil)

	// Test upper bounds
	m.pet.Hunger = maxStat
	m.feed()
	if m.pet.Hunger > maxStat {
		t.Errorf("Hunger should not exceed %d", maxStat)
	}

	m.pet.Happiness = maxStat
	m.play()
	if m.pet.Happiness > maxStat {
		t.Errorf("Happiness should not exceed %d", maxStat)
	}

	// Test lower bounds
	m.pet.Energy = minStat
	m.play()
	if m.pet.Energy < minStat {
		t.Errorf("Energy should not go below %d", minStat)
	}
}

func TestTimeBasedUpdates(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	// Save current time.Now and restore after test
	originalTimeNow := timeNow
	defer func() { timeNow = originalTimeNow }()

	// Set current time
	currentTime := time.Now().UTC()
	timeNow = func() time.Time { return currentTime }

	// Create initial pet state from 2 hours ago
	twoHoursAgo := currentTime.Add(-2 * time.Hour)
	testCfg := &TestConfig{
		InitialHunger:    maxStat, // Start at 100%
		InitialHappiness: maxStat, // Start at 100%
		InitialEnergy:    maxStat, // Start at 100%
		IsSleeping:       false,
		LastSavedTime:    twoHoursAgo, // Set last saved to 2 hours ago
	}

	// Save initial state
	pet := newPet(testCfg)
	saveState(&pet)

	// Fix the LastSaved time in the saved file
	data, err := os.ReadFile(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	var savedPet Pet
	if err := json.Unmarshal(data, &savedPet); err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	savedPet.LastSaved = twoHoursAgo
	data, err = json.MarshalIndent(savedPet, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal pet: %v", err)
	}
	if err := os.WriteFile(testConfigPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load state which will process the elapsed time
	loadedPet := loadState()

	// Verify stats decreased appropriately for 2 hours
	expectedHunger := maxStat - (2 * hungerDecreaseRate) // 2 hours * 5 per hour = 10 decrease
	if loadedPet.Hunger != expectedHunger {
		t.Errorf("Expected hunger to be %d after 2 hours, got %d", expectedHunger, loadedPet.Hunger)
	}

	expectedEnergy := maxStat - energyDecreaseRate // 2 hours = 1 energy decrease
	if loadedPet.Energy != expectedEnergy {
		t.Errorf("Expected energy to be %d after 2 hours, got %d", expectedEnergy, loadedPet.Energy)
	}

	// Happiness shouldn't decrease since hunger and energy are still above threshold
	if loadedPet.Happiness != maxStat {
		t.Errorf("Expected happiness to stay at %d, got %d", maxStat, loadedPet.Happiness)
	}
}

func TestIllnessSystem(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	t.Run("Develop illness", func(t *testing.T) {
		// Force deterministic illness check
		randFloat64 = func() float64 { return 0.05 } // Always < 0.1 illness threshold

		// Create pet with low health
		// Use fixed timestamps to ensure exact 1 hour difference
		baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).UTC()
		testCfg := &TestConfig{
			Health:        40,
			Illness:       false,
			LastSavedTime: baseTime,
		}
		pet := newPet(testCfg)
		saveState(&pet)

		// Load with exact 1 hour later time
		loadedPet := func() Pet {
			timeNow = func() time.Time { return baseTime.Add(time.Hour) }
			return loadState()
		}()
		if !loadedPet.Illness {
			t.Error("Expected pet to develop illness with low health")
		}
	})

	t.Run("Cure with medicine", func(t *testing.T) {
		// Create sick pet
		testCfg := &TestConfig{
			Health:  40,
			Illness: true,
		}
		m := initialModel(testCfg)
		m.administerMedicine()

		if m.pet.Illness {
			t.Error("Medicine should cure illness")
		}
		if m.pet.Health != 70 { // 40 + 30 medicine effect
			t.Errorf("Expected health 70 after medicine, got %d", m.pet.Health)
		}
	})

	t.Run("Prevent illness", func(t *testing.T) {
		// Create healthy pet
		testCfg := &TestConfig{
			Health:        60,
			Illness:       false,
			LastSavedTime: time.Now().Add(-1 * time.Hour),
		}
		pet := newPet(testCfg)
		saveState(&pet)

		loadedPet := loadState()
		if loadedPet.Illness {
			t.Error("Pet with health >50 shouldn't develop illness")
		}
	})

	t.Run("Auto-heal from illness", func(t *testing.T) {
		// Create sick pet that will recover
		testCfg := &TestConfig{
			Health:        40,
			Illness:       true,
			LastSavedTime: time.Now().Add(-1 * time.Hour),
		}
		pet := newPet(testCfg)
		pet.Health = 60 // Set health to safe level
		saveState(&pet)

		loadedPet := loadState()
		if loadedPet.Illness {
			t.Error("Pet should automatically recover from illness when health >= 50")
		}
	})
}

func TestGetStatus(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	t.Run("Dead status", func(t *testing.T) {
		pet := newPet(nil)
		pet.Dead = true
		if status := getStatus(pet); status != "💀 Dead" {
			t.Errorf("Expected dead status, got %s", status)
		}
	})

	t.Run("Sleeping status", func(t *testing.T) {
		pet := newPet(nil)
		pet.Sleeping = true
		if status := getStatus(pet); status != "😴 Sleeping" {
			t.Errorf("Expected sleeping status, got %s", status)
		}
	})

	t.Run("Hungry status", func(t *testing.T) {
		pet := newPet(nil)
		pet.Hunger = lowStatThreshold - 1
		if status := getStatus(pet); status != "🙀 Hungry" {
			t.Errorf("Expected hungry status, got %s", status)
		}
	})

	t.Run("Happy status", func(t *testing.T) {
		pet := newPet(nil)
		pet.Hunger = maxStat
		pet.Energy = maxStat
		pet.Happiness = maxStat
		if status := getStatus(pet); status != "😸 Happy" {
			t.Errorf("Expected happy status, got %s", status)
		}
	})
}

func TestNewPetLogging(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	pet := newPet(nil)

	if len(pet.Logs) != 1 {
		t.Fatalf("New pet should have initial log entry, got %d entries", len(pet.Logs))
	}

	firstLog := pet.Logs[0]
	if firstLog.NewStatus != "😸 Happy" {
		t.Errorf("Expected initial status '😸 Happy', got '%s'", firstLog.NewStatus)
	}
	if firstLog.Time.After(time.Now()) {
		t.Error("Initial log time should not be in the future")
	}
}

func TestStatusLogging(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	t.Run("Multiple status changes", func(t *testing.T) {
		pet := newPet(nil)
		initialStatus := pet.LastStatus

		// First change: Make pet hungry
		pet.Hunger = 20
		saveState(&pet)

		// Second change: Make pet tired
		pet.Energy = 20
		saveState(&pet)

		// Third change: Make pet sleep
		pet.Sleeping = true
		saveState(&pet)

		if len(pet.Logs) != 4 { // Initial + 3 changes
			t.Fatalf("Expected 4 log entries, got %d", len(pet.Logs))
		}

		// Verify the sequence of status changes
		expectedStatuses := []struct {
			old string
			new string
		}{
			{"", initialStatus},         // Initial status
			{initialStatus, "🙀 Hungry"},  //First change
			{"🙀 Hungry", "😾 Tired"},     // Second change
			{"😾 Tired", "😴 Sleeping"},   // Third change
		}

		for i, expected := range expectedStatuses {
			if pet.Logs[i].OldStatus != expected.old {
				t.Errorf("Log %d: Expected old status '%s', got '%s'",
					i, expected.old, pet.Logs[i].OldStatus)
			}
			if pet.Logs[i].NewStatus != expected.new {
				t.Errorf("Log %d: Expected new status '%s', got '%s'",
					i, expected.new, pet.Logs[i].NewStatus)
			}
		}
	})

	t.Run("Single status change", func(t *testing.T) {
		pet := newPet(nil)
		pet.Hunger = 20 // Should trigger hungry status
		saveState(&pet)

		if len(pet.Logs) != 2 {
			t.Fatalf("Expected 2 log entries, got %d", len(pet.Logs))
		}

		changeLog := pet.Logs[1]
		if changeLog.OldStatus != "😸 Happy" {
			t.Errorf("Expected OldStatus '😸 Happy', got '%s'", changeLog.OldStatus)
		}
		if changeLog.NewStatus != "🙀 Hungry" {
			t.Errorf("Expected NewStatus '🙀 Hungry', got '%s'", changeLog.NewStatus)
		}
	})

	t.Run("No status change", func(t *testing.T) {
		pet := newPet(nil)
		initialLogCount := len(pet.Logs)

		// No actual status change
		pet.Happiness = 95
		saveState(&pet)

		if len(pet.Logs) != initialLogCount {
			t.Error("Should not create new log entry when status doesn't change")
		}
	})

	t.Run("Loading existing state with logs", func(t *testing.T) {
		// Create pet with existing logs
		pet := newPet(nil)
		pet.Hunger = 20
		saveState(&pet)
		initialLogCount := len(pet.Logs)

		// Load state and make new change
		loadedPet := loadState()
		loadedPet.Hunger = 50 // Reset hunger above threshold
		loadedPet.Energy = 20 // Now energy is the lowest stat
		saveState(&loadedPet)

		if len(loadedPet.Logs) != initialLogCount+1 {
			t.Errorf("Should append new log entries, expected %d got %d",
				initialLogCount+1, len(loadedPet.Logs))
		}
	})
}

func TestAging(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	t.Run("Age increases over time", func(t *testing.T) {
		// Set current time
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		originalTimeNow := timeNow
		timeNow = func() time.Time { return currentTime }
		defer func() { timeNow = originalTimeNow }()

		// Create pet 5 hours ago
		fiveHoursAgo := currentTime.Add(-5 * time.Hour)
		testCfg := &TestConfig{
			LastSavedTime: fiveHoursAgo,
		}
		pet := newPet(testCfg)
		
		// Set age directly to avoid double-counting
		pet.Age = 0
		
		// Manually set the birth time in logs
		pet.Logs = []LogEntry{{
			Time:      fiveHoursAgo,
			OldStatus: "",
			NewStatus: "😸 Happy",
		}}
		saveState(&pet)

		// Fix the LastSaved time in the saved file
		data, err := os.ReadFile(testConfigPath)
		if err != nil {
			t.Fatalf("Failed to read test file: %v", err)
		}
		var savedPet Pet
		if err := json.Unmarshal(data, &savedPet); err != nil {
			t.Fatalf("Failed to parse test file: %v", err)
		}
		savedPet.LastSaved = fiveHoursAgo
		savedPet.Age = 0 // Reset age in the file
		data, err = json.MarshalIndent(savedPet, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal pet: %v", err)
		}
		if err := os.WriteFile(testConfigPath, data, 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Load state which will process elapsed time
		loadedPet := loadState()

		if loadedPet.Age != 5 {
			t.Errorf("Expected age to be 5 hours, got %d", loadedPet.Age)
		}
	})

	t.Run("Life stages transition correctly", func(t *testing.T) {
		// Set current time
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		originalTimeNow := timeNow
		timeNow = func() time.Time { return currentTime }
		defer func() { timeNow = originalTimeNow }()

		testCases := []struct {
			hours     int
			expected  int
			stageName string
		}{
			{0, 0, "Baby"},
			{47, 0, "Baby"},
			{48, 1, "Child"},
			{95, 1, "Child"},
			{96, 2, "Adult"},
			{200, 2, "Adult"}, // Should stay adult
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("%d hours = %s", tc.hours, tc.stageName), func(t *testing.T) {
				// Create a new pet with birth time set to the correct time in the past
				birthTime := currentTime.Add(time.Duration(-tc.hours) * time.Hour)
				
				// Create a pet with LastSaved = birthTime (no elapsed time yet)
				testCfg := &TestConfig{
					LastSavedTime: birthTime,
				}
				pet := newPet(testCfg)
				
				// Reset age and life stage to ensure they're calculated correctly
				pet.Age = 0
				pet.LifeStage = 0
				
				// Set birth time in logs
				pet.Logs = []LogEntry{{
					Time:      birthTime,
					OldStatus: "",
					NewStatus: "😸 Happy",
				}}
				
				// Save with these initial values
				saveState(&pet)
				
				// Modify the saved file to ensure LastSaved is exactly at birth time
				data, err := os.ReadFile(testConfigPath)
				if err != nil {
					t.Fatalf("Failed to read test file: %v", err)
				}
				var savedPet Pet
				if err := json.Unmarshal(data, &savedPet); err != nil {
					t.Fatalf("Failed to parse test file: %v", err)
				}
				savedPet.LastSaved = birthTime
				savedPet.Age = 0
				savedPet.LifeStage = 0
				data, err = json.MarshalIndent(savedPet, "", "  ")
				if err != nil {
					t.Fatalf("Failed to marshal pet: %v", err)
				}
				if err := os.WriteFile(testConfigPath, data, 0644); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
				
				// Now load the pet, which should calculate age based on elapsed time
				loadedPet := loadState()
				
				if loadedPet.Age != tc.hours {
					t.Errorf("Expected age %d, got %d", tc.hours, loadedPet.Age)
				}
				
				if loadedPet.LifeStage != tc.expected {
					t.Errorf("At %d hours: Expected life stage %d (%s), got %d",
						tc.hours, tc.expected, tc.stageName, loadedPet.LifeStage)
				}
			})
		}
	})
}

func TestDeathLogging(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	pet := newPet(nil)
	pet.Dead = true
	pet.CauseOfDeath = "Old Age"
	saveState(&pet)

	if len(pet.Logs) < 1 {
		t.Fatal("Should have death log entry")
	}

	lastLog := pet.Logs[len(pet.Logs)-1]
	if lastLog.NewStatus != "💀 Dead" {
		t.Errorf("Last log entry should be death status, got '%s'", lastLog.NewStatus)
	}
}
