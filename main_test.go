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

// mockTimeNow sets a fixed time for deterministic tests and auto-restores after test
func mockTimeNow(t *testing.T) time.Time {
	originalTimeNow := timeNow
	currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local)
	timeNow = func() time.Time { return currentTime }
	t.Cleanup(func() { timeNow = originalTimeNow })
	return currentTime
}

func TestDeathConditions(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	currentTime := mockTimeNow(t)

	// Create pet that has been critical for 13 hours (exceeds 12h threshold)
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

func TestNaturalDeathFromOldAge(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	currentTime := mockTimeNow(t)

	// Save original randFloat64 and restore after test
	originalRandFloat64 := randFloat64
	defer func() { randFloat64 = originalRandFloat64 }()

	// Create a healthy but old pet (168+ hours)
	birthTime := currentTime.Add(-200 * time.Hour)

	testCfg := &TestConfig{
		InitialHunger:    100,
		InitialHappiness: 100,
		InitialEnergy:    100,
		Health:           100,
		LastSavedTime:    birthTime,
	}
	pet := newPet(testCfg)
	saveState(&pet)

	// Fix LastSaved time in file to make pet 200 hours old
	data, err := os.ReadFile(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	var savedPet Pet
	if err := json.Unmarshal(data, &savedPet); err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	savedPet.LastSaved = birthTime
	data, err = json.MarshalIndent(savedPet, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal pet: %v", err)
	}
	if err := os.WriteFile(testConfigPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test old age death triggers
	randFloat64 = func() float64 { return 0.0 } // Always trigger death
	loadedPet := loadState()

	if !loadedPet.Dead {
		t.Error("Expected old pet (200h) to die of old age")
	}
	if loadedPet.CauseOfDeath != "Old Age" {
		t.Errorf("Expected death cause 'Old Age', got '%s'", loadedPet.CauseOfDeath)
	}

	// Test old age death doesn't trigger with high random value
	cleanup() // Reset test file
	cleanup = setupTestFile(t)

	pet = newPet(testCfg)
	saveState(&pet)

	// Fix LastSaved time again
	data, _ = os.ReadFile(testConfigPath)
	json.Unmarshal(data, &savedPet)
	savedPet.LastSaved = birthTime
	data, _ = json.MarshalIndent(savedPet, "", "  ")
	os.WriteFile(testConfigPath, data, 0644)

	randFloat64 = func() float64 { return 1.0 } // Never trigger death
	loadedPet = loadState()

	if loadedPet.Dead {
		t.Error("Expected old pet not to die when random value is high")
	}
}

func TestDeathCausePriority(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	currentTime := mockTimeNow(t)
	criticalStart := currentTime.Add(-13 * time.Hour)

	t.Run("Starvation takes priority over Sickness", func(t *testing.T) {
		testCfg := &TestConfig{
			InitialHunger:    0, // Starving
			InitialHappiness: 50,
			InitialEnergy:    50,
			Health:           20,
			Illness:          true, // Also sick
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
			t.Error("Expected pet to be dead")
		}
		if loadedPet.CauseOfDeath != "Starvation" {
			t.Errorf("Expected 'Starvation' (not 'Sickness'), got '%s'", loadedPet.CauseOfDeath)
		}
	})

	t.Run("Sickness when not starving", func(t *testing.T) {
		cleanup() // Reset
		cleanup = setupTestFile(t)

		testCfg := &TestConfig{
			InitialHunger:    70, // High enough to not hit 0 after 13h (13*5=65 decrease)
			InitialHappiness: 5,
			InitialEnergy:    5,
			Health:           5,
			Illness:          true,
			LastSavedTime:    criticalStart,
		}
		pet := newPet(testCfg)
		pet.CriticalStartTime = &criticalStart
		pet.Traits = []Trait{} // Clear traits for predictable test results
		saveState(&pet)

		// Fix LastSaved time in file
		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = criticalStart
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		if !loadedPet.Dead {
			t.Error("Expected pet to be dead")
		}
		if loadedPet.CauseOfDeath != "Sickness" {
			t.Errorf("Expected 'Sickness', got '%s'", loadedPet.CauseOfDeath)
		}
	})

	t.Run("Neglect when all stats critical", func(t *testing.T) {
		cleanup() // Reset
		cleanup = setupTestFile(t)

		testCfg := &TestConfig{
			InitialHunger:    70, // High enough to not hit 0 (13*5=65 decrease)
			InitialHappiness: 5,
			InitialEnergy:    50, // High enough to not hit 0 (13/2=6.5 decreases * 5 = 32.5)
			Health:           5,
			Illness:          false, // Not sick
			LastSavedTime:    criticalStart,
		}
		pet := newPet(testCfg)
		pet.CriticalStartTime = &criticalStart
		pet.Traits = []Trait{} // Clear traits for predictable test results
		saveState(&pet)

		// Fix LastSaved time in file
		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = criticalStart
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		if !loadedPet.Dead {
			t.Error("Expected pet to be dead")
		}
		if loadedPet.CauseOfDeath != "Neglect" {
			t.Errorf("Expected 'Neglect', got '%s'", loadedPet.CauseOfDeath)
		}
	})
}

func TestCriticalStateRecovery(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	currentTime := mockTimeNow(t)

	// Create pet NOT in critical state initially
	// We'll manually set CriticalStartTime to simulate it was in critical state before
	twoHoursAgo := currentTime.Add(-2 * time.Hour)
	testCfg := &TestConfig{
		InitialHunger:    50,  // Above critical
		InitialHappiness: 50,  // Above critical
		InitialEnergy:    50,  // Above critical
		Health:           50,  // Above critical
		LastSavedTime:    twoHoursAgo,
	}

	pet := newPet(testCfg)
	// Manually set critical start time to simulate pet WAS in critical state
	oneHourAgo := currentTime.Add(-1 * time.Hour)
	pet.CriticalStartTime = &oneHourAgo
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
	savedPet.LastSaved = twoHoursAgo
	data, err = json.MarshalIndent(savedPet, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal pet: %v", err)
	}
	if err := os.WriteFile(testConfigPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Verify pet has CriticalStartTime set
	if pet.CriticalStartTime == nil {
		t.Error("Pet should have CriticalStartTime set for test")
	}

	// Load state - pet should recover from critical state
	// After 2 hours: Hunger=50-10=40, Happiness=50, Energy=50-5=45, Health=50
	// All above thresholds: Health>20, Hunger>=10, Happiness>=10, Energy>=10
	loadedPet := loadState()

	// Verify CriticalStartTime has been reset
	if loadedPet.CriticalStartTime != nil {
		t.Errorf("CriticalStartTime should be nil after recovery from critical state, got %v", loadedPet.CriticalStartTime)
	}

	// Verify pet is not dead
	if loadedPet.Dead {
		t.Error("Pet should not be dead after recovery")
	}

	// Verify all stats are above critical thresholds
	if loadedPet.Health <= 20 {
		t.Errorf("Health should be > 20, got %d", loadedPet.Health)
	}
	if loadedPet.Hunger < 10 {
		t.Errorf("Hunger should be >= 10, got %d", loadedPet.Hunger)
	}
	if loadedPet.Happiness < 10 {
		t.Errorf("Happiness should be >= 10, got %d", loadedPet.Happiness)
	}
	if loadedPet.Energy < 10 {
		t.Errorf("Energy should be >= 10, got %d", loadedPet.Energy)
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

	currentTime := mockTimeNow(t)

	testCfg := &TestConfig{
		InitialHunger:    50,  // Start with lower hunger
		InitialHappiness: 50,  // Start with lower happiness
		InitialEnergy:    100, // Start with full energy
		Health:           80,  // Start with suboptimal health
		IsSleeping:       false,
		LastSavedTime:    currentTime,
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

func TestSleepingPetStaysAsleepAtFullEnergy(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	currentTime := mockTimeNow(t)

	// Create sleeping pet with low energy
	oneHourAgo := currentTime.Add(-1 * time.Hour)
	testCfg := &TestConfig{
		InitialHunger:    100,
		InitialHappiness: 100,
		InitialEnergy:    90, // Below max
		Health:           100,
		IsSleeping:       true,
		LastSavedTime:    oneHourAgo,
	}

	pet := newPet(testCfg)
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
	savedPet.LastSaved = oneHourAgo
	data, err = json.MarshalIndent(savedPet, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal pet: %v", err)
	}
	if err := os.WriteFile(testConfigPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load state - after 1 hour of sleeping, energy should be 100
	loadedPet := loadState()

	if loadedPet.Energy != 100 {
		t.Errorf("Expected energy to be 100 after 1 hour of sleep, got %d", loadedPet.Energy)
	}

	// Critical test: Pet should STAY ASLEEP even though energy is at max
	if !loadedPet.Sleeping {
		t.Error("Pet should stay asleep even at full energy (no auto-wake)")
	}
}

func TestTimeBasedUpdates(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	// Save current time.Now and restore after test
	originalTimeNow := timeNow
	defer func() { timeNow = originalTimeNow }()

	// Save original randFloat64 and restore after test
	originalRandFloat64 := randFloat64
	defer func() { randFloat64 = originalRandFloat64 }()

	// Set current time to noon local time (during active hours for Night Owl chronotype)
	currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local)
	timeNow = func() time.Time { return currentTime }

	// Prevent random illness from making test non-deterministic
	randFloat64 = func() float64 { return 1.0 }

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
	pet.Chronotype = ChronotypeNightOwl // Set chronotype where noon is in active hours (10am-2am)
	pet.Traits = []Trait{}              // Clear traits for predictable test results
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

	currentTime := mockTimeNow(t)

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
		pet.Traits = []Trait{} // Clear traits for predictable results
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
			LastSavedTime: currentTime.Add(-1 * time.Hour),
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
			LastSavedTime: currentTime.Add(-1 * time.Hour),
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
		if status := getStatus(pet); status != "💀" {
			t.Errorf("Expected 💀, got %s", status)
		}
	})

	t.Run("Sleeping status", func(t *testing.T) {
		pet := newPet(nil)
		pet.Sleeping = true
		if status := getStatus(pet); status != "😴" {
			t.Errorf("Expected 😴, got %s", status)
		}
	})

	t.Run("Hungry status (awake)", func(t *testing.T) {
		pet := newPet(nil)
		pet.Hunger = lowStatThreshold - 1
		// Activity (awake) + Feeling (hungry)
		if status := getStatus(pet); status != "😸🙀" {
			t.Errorf("Expected 😸🙀, got %s", status)
		}
	})

	t.Run("Hungry status (sleeping)", func(t *testing.T) {
		pet := newPet(nil)
		pet.Hunger = lowStatThreshold - 1
		pet.Sleeping = true
		// Activity (sleeping) + Feeling (hungry)
		if status := getStatus(pet); status != "😴🙀" {
			t.Errorf("Expected 😴🙀, got %s", status)
		}
	})

	t.Run("Happy status", func(t *testing.T) {
		pet := newPet(nil)
		pet.Hunger = maxStat
		pet.Energy = maxStat
		pet.Happiness = maxStat
		// Activity (awake) + no feeling (all good)
		if status := getStatus(pet); status != "😸" {
			t.Errorf("Expected 😸, got %s", status)
		}
	})

	t.Run("Event with critical stat", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		originalTimeNow := timeNow
		timeNow = func() time.Time { return currentTime }
		defer func() { timeNow = originalTimeNow }()

		pet := newPet(nil)
		pet.Hunger = lowStatThreshold - 1 // Critical hunger
		pet.CurrentEvent = &Event{
			Type:      EventChasing,
			StartTime: currentTime,
			ExpiresAt: currentTime.Add(10 * time.Minute),
			Responded: false,
		}
		// Should show event emoji + hungry feeling
		if status := getStatus(pet); status != "🦋🙀" {
			t.Errorf("Expected 🦋🙀 (event + hungry), got %s", status)
		}
	})

	t.Run("Event without critical stat", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		originalTimeNow := timeNow
		timeNow = func() time.Time { return currentTime }
		defer func() { timeNow = originalTimeNow }()

		pet := newPet(nil)
		pet.CurrentEvent = &Event{
			Type:      EventSinging,
			StartTime: currentTime,
			ExpiresAt: currentTime.Add(5 * time.Minute),
			Responded: false,
		}
		// Should show just event emoji
		if status := getStatus(pet); status != "🎵" {
			t.Errorf("Expected 🎵 (event only), got %s", status)
		}
	})

	t.Run("Sleeping with critical stat", func(t *testing.T) {
		pet := newPet(nil)
		pet.Sleeping = true
		pet.Energy = lowStatThreshold - 1 // Critical energy
		// Activity (sleeping) + Feeling (tired)
		if status := getStatus(pet); status != "😴😾" {
			t.Errorf("Expected 😴😾, got %s", status)
		}
	})

	t.Run("Drowsy status when energy between thresholds", func(t *testing.T) {
		pet := newPet(nil)
		pet.Energy = 35 // Above critical (30) but below drowsy (40)
		pet.Sleeping = false
		// Should show awake + drowsy
		if status := getStatus(pet); status != "😸🥱" {
			t.Errorf("Expected 😸🥱 (drowsy), got %s", status)
		}
	})

	t.Run("No drowsy when sleeping", func(t *testing.T) {
		pet := newPet(nil)
		pet.Energy = 35 // Below drowsy threshold
		pet.Sleeping = true
		// Should not show drowsy when sleeping
		if status := getStatus(pet); status != "😴" {
			t.Errorf("Expected 😴 (sleeping, no drowsy), got %s", status)
		}
	})

	t.Run("Sad status lowest priority", func(t *testing.T) {
		pet := newPet(nil)
		pet.Happiness = lowStatThreshold - 1 // Low happiness
		// Activity (awake) + Feeling (sad)
		if status := getStatus(pet); status != "😸😿" {
			t.Errorf("Expected 😸😿, got %s", status)
		}
	})

	t.Run("Sick status when health is lowest", func(t *testing.T) {
		pet := newPet(nil)
		pet.Health = lowStatThreshold - 1 // Low health
		// Activity (awake) + Feeling (sick)
		if status := getStatus(pet); status != "😸🤢" {
			t.Errorf("Expected 😸🤢, got %s", status)
		}
	})

	t.Run("Lowest stat determines feeling", func(t *testing.T) {
		pet := newPet(nil)
		pet.Hunger = 25    // Low
		pet.Happiness = 20 // Lowest
		pet.Energy = 28    // Low
		pet.Health = 26    // Low
		// Happiness is lowest, so should show sad
		if status := getStatus(pet); status != "😸😿" {
			t.Errorf("Expected 😸😿 (sad for lowest happiness), got %s", status)
		}
	})

	t.Run("Responded event shows normal activity", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		originalTimeNow := timeNow
		timeNow = func() time.Time { return currentTime }
		defer func() { timeNow = originalTimeNow }()

		pet := newPet(nil)
		pet.CurrentEvent = &Event{
			Type:      EventChasing,
			StartTime: currentTime,
			ExpiresAt: currentTime.Add(10 * time.Minute),
			Responded: true, // Already responded
		}
		// Should show normal awake status since event is responded
		if status := getStatus(pet); status != "😸" {
			t.Errorf("Expected 😸 (responded event shows normal), got %s", status)
		}
	})
}

func TestNewPetLogging(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	currentTime := mockTimeNow(t)

	pet := newPet(nil)

	if len(pet.Logs) != 1 {
		t.Fatalf("New pet should have initial log entry, got %d entries", len(pet.Logs))
	}

	firstLog := pet.Logs[0]
	if firstLog.NewStatus != "😸" {
		t.Errorf("Expected initial status '😸', got '%s'", firstLog.NewStatus)
	}
	if firstLog.Time.After(currentTime) {
		t.Error("Initial log time should not be in the future")
	}
}

func TestStatusLogging(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	t.Run("Multiple status changes", func(t *testing.T) {
		pet := newPet(nil)
		initialStatus := pet.LastStatus

		// First change: Make pet hungry (lowest stat)
		pet.Hunger = 20
		saveState(&pet)

		// Second change: Make pet more tired than hungry (new lowest stat)
		pet.Energy = 15
		saveState(&pet)

		// Third change: Restore stats and make pet sleep
		pet.Hunger = 50
		pet.Energy = 50
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
			{"", initialStatus},           // Initial status
			{initialStatus, "😸🙀"},       // First change: awake + hungry
			{"😸🙀", "😸😾"},              // Second change: awake + tired (lower than hungry)
			{"😸😾", "😴"},                // Third change: sleeping, no critical stats
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
		if changeLog.OldStatus != "😸" {
			t.Errorf("Expected OldStatus '😸', got '%s'", changeLog.OldStatus)
		}
		if changeLog.NewStatus != "😸🙀" {
			t.Errorf("Expected NewStatus '😸🙀', got '%s'", changeLog.NewStatus)
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
	if lastLog.NewStatus != "💀" {
		t.Errorf("Last log entry should be death status '💀', got '%s'", lastLog.NewStatus)
	}
}

func TestStatCalculationPrecision(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	// Save original functions and restore after test
	originalTimeNow := timeNow
	originalRandFloat64 := randFloat64
	defer func() {
		timeNow = originalTimeNow
		randFloat64 = originalRandFloat64
	}()

	// Prevent random illness/events from interfering
	randFloat64 = func() float64 { return 1.0 }

	t.Run("Short elapsed time updates stats correctly", func(t *testing.T) {
		// This tests the floating-point fix - previously int(elapsed.Minutes()) truncated small intervals
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		threeSecondsAgo := currentTime.Add(-3 * time.Second) // Typical tmux update interval
		timeNow = func() time.Time { return currentTime }

		testCfg := &TestConfig{
			InitialHunger:    100,
			InitialHappiness: 100,
			InitialEnergy:    100,
			Health:           100,
			LastSavedTime:    threeSecondsAgo,
		}
		pet := newPet(testCfg)
		saveState(&pet)

		// Fix LastSaved time in file
		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = threeSecondsAgo
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		// 3 seconds = 0.000833 hours
		// With 5/hr hunger rate: 0.000833 * 5 = 0.004 ≈ 0 (truncated)
		// Stats should not change significantly for such short interval
		if loadedPet.Hunger != 100 {
			t.Errorf("Expected hunger 100 for 3-second interval, got %d", loadedPet.Hunger)
		}
	})

	t.Run("One hour updates stats correctly", func(t *testing.T) {
		// Use local time noon which is active hours for Night Owl (10am-2am)
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local)
		oneHourAgo := currentTime.Add(-1 * time.Hour)
		timeNow = func() time.Time { return currentTime }

		testCfg := &TestConfig{
			InitialHunger:    100,
			InitialHappiness: 100,
			InitialEnergy:    100,
			Health:           100,
			IsSleeping:       false,
			LastSavedTime:    oneHourAgo,
		}
		pet := newPet(testCfg)
		pet.Chronotype = ChronotypeNightOwl // Noon is active hours for Night Owl
		pet.Traits = []Trait{}              // Clear traits for predictable results
		saveState(&pet)

		// Fix LastSaved time in file
		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = oneHourAgo
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		// 1 hour awake: hunger -5, energy -2 (every 2 hours so ~2), happiness unchanged
		expectedHunger := 100 - hungerDecreaseRate // 100 - 5 = 95
		if loadedPet.Hunger != expectedHunger {
			t.Errorf("Expected hunger %d, got %d", expectedHunger, loadedPet.Hunger)
		}

		// Energy decreases every 2 hours, so 1 hour = 0.5 cycles = 2 energy loss
		expectedEnergy := 100 - (energyDecreaseRate / 2) // 100 - 2 = 98
		if loadedPet.Energy != expectedEnergy {
			t.Errorf("Expected energy %d, got %d", expectedEnergy, loadedPet.Energy)
		}
	})

	t.Run("Sleep recovery uses floating point correctly", func(t *testing.T) {
		// Use local time noon which is active hours for Night Owl (no recovery boost)
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local)
		thirtyMinutesAgo := currentTime.Add(-30 * time.Minute)
		timeNow = func() time.Time { return currentTime }

		testCfg := &TestConfig{
			InitialHunger:    100,
			InitialHappiness: 100,
			InitialEnergy:    50, // Start with 50 energy
			Health:           100,
			IsSleeping:       true,
			LastSavedTime:    thirtyMinutesAgo,
		}
		pet := newPet(testCfg)
		pet.Chronotype = ChronotypeNightOwl // Noon is active hours (no sleep recovery boost)
		saveState(&pet)

		// Fix LastSaved time in file
		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = thirtyMinutesAgo
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		// 30 minutes = 0.5 hours
		// Energy recovery: 0.5 * 10 = 5
		// Hunger (sleeping): 0.5 * 3 = 1.5 → 1
		expectedEnergy := min(50+5, maxStat) // 55
		if loadedPet.Energy != expectedEnergy {
			t.Errorf("Expected energy %d after 30min sleep, got %d", expectedEnergy, loadedPet.Energy)
		}

		expectedHunger := 100 - 1 // 0.5 * 3 = 1.5 truncated to 1
		if loadedPet.Hunger != expectedHunger {
			t.Errorf("Expected hunger %d, got %d", expectedHunger, loadedPet.Hunger)
		}
	})

	t.Run("Happiness decay when stats low", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		twoHoursAgo := currentTime.Add(-2 * time.Hour)
		timeNow = func() time.Time { return currentTime }

		testCfg := &TestConfig{
			InitialHunger:    20, // Below threshold (30)
			InitialHappiness: 100,
			InitialEnergy:    100,
			Health:           100,
			IsSleeping:       false,
			LastSavedTime:    twoHoursAgo,
		}
		pet := newPet(testCfg)
		saveState(&pet)

		// Fix LastSaved time in file
		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = twoHoursAgo
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		// 2 hours with low hunger: happiness decreases 2 * 2 = 4
		expectedHappiness := 100 - (2 * happinessDecreaseRate) // 100 - 4 = 96
		if loadedPet.Happiness != expectedHappiness {
			t.Errorf("Expected happiness %d, got %d", expectedHappiness, loadedPet.Happiness)
		}
	})

	t.Run("Health decay with critically low stats", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		threeHoursAgo := currentTime.Add(-3 * time.Hour)
		timeNow = func() time.Time { return currentTime }

		testCfg := &TestConfig{
			InitialHunger:    10, // Below 15 (critical)
			InitialHappiness: 100,
			InitialEnergy:    100,
			Health:           100,
			IsSleeping:       false,
			LastSavedTime:    threeHoursAgo,
		}
		pet := newPet(testCfg)
		pet.Traits = []Trait{} // Clear traits for predictable results
		saveState(&pet)

		// Fix LastSaved time in file
		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = threeHoursAgo
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		// 3 hours with critically low hunger: health decreases 3 * 2 = 6
		expectedHealth := 100 - (3 * healthDecreaseRate) // 100 - 6 = 94
		if loadedPet.Health != expectedHealth {
			t.Errorf("Expected health %d, got %d", expectedHealth, loadedPet.Health)
		}
	})
}

func TestActionRefusal(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	// Save original functions and restore after test
	originalTimeNow := timeNow
	defer func() { timeNow = originalTimeNow }()

	currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return currentTime }

	t.Run("Feed refused when too full", func(t *testing.T) {
		testCfg := &TestConfig{
			InitialHunger:    95, // Above 90 threshold
			InitialHappiness: 50,
			InitialEnergy:    50,
			Health:           50,
			LastSavedTime:    currentTime,
		}
		m := initialModel(testCfg)
		originalHunger := m.pet.Hunger

		m.feed()

		// Hunger should not change
		if m.pet.Hunger != originalHunger {
			t.Errorf("Hunger should not change when too full, was %d now %d", originalHunger, m.pet.Hunger)
		}
		// Should have refusal message
		if m.message == "" {
			t.Error("Should show refusal message")
		}
	})

	t.Run("Feed succeeds when hungry", func(t *testing.T) {
		testCfg := &TestConfig{
			InitialHunger:    50, // Below 90 threshold
			InitialHappiness: 50,
			InitialEnergy:    50,
			Health:           50,
			LastSavedTime:    currentTime,
		}
		m := initialModel(testCfg)

		m.feed()

		if m.pet.Hunger <= 50 {
			t.Errorf("Hunger should increase after feeding, got %d", m.pet.Hunger)
		}
	})

	t.Run("Play refused when too tired", func(t *testing.T) {
		testCfg := &TestConfig{
			InitialHunger:    50,
			InitialHappiness: 50,
			InitialEnergy:    autoSleepThreshold - 1, // Below threshold (19)
			Health:           50,
			LastSavedTime:    currentTime,
		}
		m := initialModel(testCfg)
		originalHappiness := m.pet.Happiness

		m.play()

		// Happiness should not change
		if m.pet.Happiness != originalHappiness {
			t.Error("Happiness should not change when too tired to play")
		}
		// Should have refusal message
		if m.message == "" {
			t.Error("Should show refusal message")
		}
	})

	t.Run("Play refused when lazy mood and low energy", func(t *testing.T) {
		testCfg := &TestConfig{
			InitialHunger:    50,
			InitialHappiness: 50,
			InitialEnergy:    45, // Below 50 threshold for lazy mood check
			Health:           50,
			LastSavedTime:    currentTime,
		}
		m := initialModel(testCfg)
		m.pet.Mood = "lazy"
		originalHappiness := m.pet.Happiness

		m.play()

		// Happiness should not change
		if m.pet.Happiness != originalHappiness {
			t.Error("Lazy pet with low energy should refuse to play")
		}
		if m.message == "" {
			t.Error("Should show refusal message for lazy pet")
		}
	})

	t.Run("Lazy pet plays when energy high enough", func(t *testing.T) {
		testCfg := &TestConfig{
			InitialHunger:    50,
			InitialHappiness: 50,
			InitialEnergy:    60, // Above 50 threshold
			Health:           50,
			LastSavedTime:    currentTime,
		}
		m := initialModel(testCfg)
		m.pet.Mood = "lazy"

		m.play()

		// Happiness should increase
		if m.pet.Happiness <= 50 {
			t.Error("Lazy pet should play when energy is high enough")
		}
	})

	t.Run("Play succeeds with playful mood message", func(t *testing.T) {
		testCfg := &TestConfig{
			InitialHunger:    50,
			InitialHappiness: 50,
			InitialEnergy:    60,
			Health:           50,
			LastSavedTime:    currentTime,
		}
		m := initialModel(testCfg)
		m.pet.Mood = "playful"

		m.play()

		if m.pet.Happiness <= 50 {
			t.Error("Play should increase happiness")
		}
		// Should have happy message
		if m.message == "" {
			t.Error("Should show success message")
		}
	})

	t.Run("Feed clears auto-sleep", func(t *testing.T) {
		sleepTime := currentTime.Add(-1 * time.Hour)
		testCfg := &TestConfig{
			InitialHunger:    50,
			InitialHappiness: 50,
			InitialEnergy:    50,
			Health:           50,
			IsSleeping:       true,
			LastSavedTime:    currentTime,
		}
		m := initialModel(testCfg)
		m.pet.AutoSleepTime = &sleepTime

		m.feed()

		if m.pet.Sleeping {
			t.Error("Feeding should wake pet")
		}
		if m.pet.AutoSleepTime != nil {
			t.Error("Feeding should clear AutoSleepTime")
		}
	})

	t.Run("Play clears auto-sleep", func(t *testing.T) {
		sleepTime := currentTime.Add(-1 * time.Hour)
		testCfg := &TestConfig{
			InitialHunger:    50,
			InitialHappiness: 50,
			InitialEnergy:    50,
			Health:           50,
			IsSleeping:       true,
			LastSavedTime:    currentTime,
		}
		m := initialModel(testCfg)
		m.pet.AutoSleepTime = &sleepTime

		m.play()

		if m.pet.Sleeping {
			t.Error("Playing should wake pet")
		}
		if m.pet.AutoSleepTime != nil {
			t.Error("Playing should clear AutoSleepTime")
		}
	})
}

func TestLifeEvents(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	// Save original functions and restore after test
	originalTimeNow := timeNow
	originalRandFloat64 := randFloat64
	defer func() {
		timeNow = originalTimeNow
		randFloat64 = originalRandFloat64
	}()

	t.Run("Event triggers when conditions met", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }
		// High chance roll to trigger event
		randFloat64 = func() float64 { return 0.01 } // Very low = high chance

		pet := newPet(nil)
		pet.Sleeping = false
		pet.Energy = 50
		pet.Mood = "playful"
		pet.CurrentEvent = nil

		triggerRandomEvent(&pet)

		if pet.CurrentEvent == nil {
			t.Error("Expected event to trigger")
		}
	})

	t.Run("No event trigger when one is active", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }
		randFloat64 = func() float64 { return 0.01 }

		pet := newPet(nil)
		// Set up existing active event
		existingEvent := &Event{
			Type:      EventChasing,
			StartTime: currentTime,
			ExpiresAt: currentTime.Add(10 * time.Minute),
			Responded: false,
		}
		pet.CurrentEvent = existingEvent

		triggerRandomEvent(&pet)

		// Should still be the same event
		if pet.CurrentEvent.Type != EventChasing {
			t.Error("Should not replace active event")
		}
	})

	t.Run("Expired ignored event applies consequences", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }
		// High roll to prevent new event
		randFloat64 = func() float64 { return 0.99 }

		pet := newPet(nil)
		pet.Happiness = 50 // Will lose 15 from scared event
		// Set up expired, unresponded scared event
		expiredEvent := &Event{
			Type:      EventScared,
			StartTime: currentTime.Add(-10 * time.Minute),
			ExpiresAt: currentTime.Add(-5 * time.Minute), // Expired
			Responded: false,
		}
		pet.CurrentEvent = expiredEvent

		triggerRandomEvent(&pet)

		// Scared event penalty: -15 happiness
		if pet.Happiness != 35 {
			t.Errorf("Expected happiness 35 after ignored scared event, got %d", pet.Happiness)
		}
		// Event should be logged as ignored
		if len(pet.EventLog) == 0 {
			t.Error("Event should be logged")
		}
		if !pet.EventLog[len(pet.EventLog)-1].WasIgnored {
			t.Error("Event should be marked as ignored")
		}
	})

	t.Run("Respond to event gives reward", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }

		pet := newPet(nil)
		pet.Happiness = 50
		// Set up active cuddles event
		pet.CurrentEvent = &Event{
			Type:      EventCuddles,
			StartTime: currentTime,
			ExpiresAt: currentTime.Add(10 * time.Minute),
			Responded: false,
		}

		result := pet.respondToEvent()

		// Cuddles gives +25 happiness, +5 energy
		if pet.Happiness != 75 {
			t.Errorf("Expected happiness 75 after cuddles, got %d", pet.Happiness)
		}
		if result == "" {
			t.Error("Expected response message")
		}
		if !pet.CurrentEvent.Responded {
			t.Error("Event should be marked as responded")
		}
		if len(pet.EventLog) == 0 {
			t.Error("Event should be logged")
		}
		if pet.EventLog[len(pet.EventLog)-1].WasIgnored {
			t.Error("Event should not be marked as ignored")
		}
	})

	t.Run("Cannot respond to already responded event", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }

		pet := newPet(nil)
		pet.Happiness = 50
		pet.CurrentEvent = &Event{
			Type:      EventCuddles,
			StartTime: currentTime,
			ExpiresAt: currentTime.Add(10 * time.Minute),
			Responded: true, // Already responded
		}

		result := pet.respondToEvent()

		if result != "" {
			t.Error("Should not be able to respond to already responded event")
		}
		// Happiness should not change
		if pet.Happiness != 50 {
			t.Error("Happiness should not change for already responded event")
		}
	})

	t.Run("Event log limited to 20 entries", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }

		pet := newPet(nil)
		// Fill event log with 25 entries
		for i := 0; i < 25; i++ {
			pet.EventLog = append(pet.EventLog, EventLogEntry{
				Type:       EventChasing,
				Time:       currentTime,
				WasIgnored: false,
			})
		}

		// Respond to a new event to trigger log cleanup
		pet.CurrentEvent = &Event{
			Type:      EventCuddles,
			StartTime: currentTime,
			ExpiresAt: currentTime.Add(10 * time.Minute),
			Responded: false,
		}
		pet.respondToEvent()

		if len(pet.EventLog) > 20 {
			t.Errorf("Event log should be limited to 20 entries, got %d", len(pet.EventLog))
		}
	})

	t.Run("Dead pet gets no events", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }
		randFloat64 = func() float64 { return 0.01 } // Would trigger

		pet := newPet(nil)
		pet.Dead = true
		pet.CurrentEvent = nil

		triggerRandomEvent(&pet)

		if pet.CurrentEvent != nil {
			t.Error("Dead pet should not get events")
		}
	})

	t.Run("getEventDisplay returns correct info for active event", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }

		pet := newPet(nil)
		pet.CurrentEvent = &Event{
			Type:      EventChasing,
			StartTime: currentTime,
			ExpiresAt: currentTime.Add(10 * time.Minute),
			Responded: false,
		}

		emoji, msg, hasEvent := pet.getEventDisplay()

		if !hasEvent {
			t.Error("Expected hasEvent to be true")
		}
		if emoji != "🦋" {
			t.Errorf("Expected butterfly emoji, got %s", emoji)
		}
		if msg != "chasing a butterfly!" {
			t.Errorf("Expected chasing message, got %s", msg)
		}
	})

	t.Run("getEventDisplay returns false for expired event", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }

		pet := newPet(nil)
		pet.CurrentEvent = &Event{
			Type:      EventChasing,
			StartTime: currentTime.Add(-15 * time.Minute),
			ExpiresAt: currentTime.Add(-5 * time.Minute), // Expired
			Responded: false,
		}

		_, _, hasEvent := pet.getEventDisplay()

		if hasEvent {
			t.Error("Expected hasEvent to be false for expired event")
		}
	})
}

func TestAutonomousBehavior(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	// Save original functions and restore after test
	originalTimeNow := timeNow
	originalRandFloat64 := randFloat64
	defer func() {
		timeNow = originalTimeNow
		randFloat64 = originalRandFloat64
	}()

	t.Run("Auto-sleep when energy critical", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }

		pet := newPet(nil)
		pet.Energy = autoSleepThreshold // At threshold (20)
		pet.Sleeping = false

		applyAutonomousBehavior(&pet)

		if !pet.Sleeping {
			t.Error("Pet should auto-sleep when energy <= autoSleepThreshold")
		}
		if pet.AutoSleepTime == nil {
			t.Error("AutoSleepTime should be set when auto-sleeping")
		}
	})

	t.Run("No auto-sleep above threshold", func(t *testing.T) {
		// Use local time 12:00 which is active hours for Night Owl (10am-2am)
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local)
		timeNow = func() time.Time { return currentTime }

		pet := newPet(nil)
		pet.Chronotype = ChronotypeNightOwl // Ensure 12:00 local is in active hours
		pet.Energy = autoSleepThreshold + 1 // Above threshold
		pet.Sleeping = false

		applyAutonomousBehavior(&pet)

		if pet.Sleeping {
			t.Error("Pet should not auto-sleep when energy > autoSleepThreshold")
		}
	})

	t.Run("Auto-wake after minimum sleep with restored energy", func(t *testing.T) {
		// Use local time 12:00 which is active hours for Night Owl (10am-2am)
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local)
		sleepStartTime := currentTime.Add(-7 * time.Hour) // 7 hours ago (> minSleepDuration of 6)
		timeNow = func() time.Time { return currentTime }

		pet := newPet(nil)
		pet.Chronotype = ChronotypeNightOwl // Ensure 12:00 local is in active hours
		pet.Sleeping = true
		pet.AutoSleepTime = &sleepStartTime
		pet.Energy = autoWakeEnergy // Energy restored (80)

		applyAutonomousBehavior(&pet)

		if pet.Sleeping {
			t.Error("Pet should auto-wake after minimum sleep duration with restored energy")
		}
		if pet.AutoSleepTime != nil {
			t.Error("AutoSleepTime should be cleared after waking")
		}
	})

	t.Run("No auto-wake before minimum sleep duration", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		sleepStartTime := currentTime.Add(-4 * time.Hour) // Only 4 hours (< minSleepDuration of 6)
		timeNow = func() time.Time { return currentTime }

		pet := newPet(nil)
		pet.Sleeping = true
		pet.AutoSleepTime = &sleepStartTime
		pet.Energy = maxStat // Full energy

		applyAutonomousBehavior(&pet)

		if !pet.Sleeping {
			t.Error("Pet should not wake before minimum sleep duration")
		}
	})

	t.Run("Force wake after maximum sleep duration", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		sleepStartTime := currentTime.Add(-9 * time.Hour) // 9 hours (> maxSleepDuration of 8)
		timeNow = func() time.Time { return currentTime }

		pet := newPet(nil)
		pet.Sleeping = true
		pet.AutoSleepTime = &sleepStartTime
		pet.Energy = 50 // Energy not fully restored

		applyAutonomousBehavior(&pet)

		if pet.Sleeping {
			t.Error("Pet should force wake after maximum sleep duration")
		}
	})

	t.Run("Mood initialization", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }
		randFloat64 = func() float64 { return 0.5 } // Deterministic

		pet := newPet(nil)
		pet.Mood = "" // Unset mood

		applyAutonomousBehavior(&pet)

		if pet.Mood == "" {
			t.Error("Mood should be initialized if empty")
		}
		if pet.MoodExpiresAt == nil {
			t.Error("MoodExpiresAt should be set")
		}
	})

	t.Run("Mood changes when expired", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		expiredTime := currentTime.Add(-1 * time.Hour) // Expired 1 hour ago
		timeNow = func() time.Time { return currentTime }
		randFloat64 = func() float64 { return 0.75 } // Will trigger "playful" for rested/happy pet

		pet := newPet(nil)
		pet.Mood = "normal"
		pet.MoodExpiresAt = &expiredTime

		applyAutonomousBehavior(&pet)

		if pet.MoodExpiresAt == nil || !pet.MoodExpiresAt.After(currentTime) {
			t.Error("MoodExpiresAt should be updated to future time")
		}
	})

	t.Run("Tired pet more likely to be lazy", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }
		randFloat64 = func() float64 { return 0.3 } // < 0.6 = lazy when tired

		pet := newPet(nil)
		pet.Energy = drowsyThreshold - 1 // Below drowsy threshold
		pet.Mood = ""
		pet.MoodExpiresAt = nil

		applyAutonomousBehavior(&pet)

		if pet.Mood != "lazy" {
			t.Errorf("Expected 'lazy' mood for tired pet with low roll, got '%s'", pet.Mood)
		}
	})

	t.Run("Dead pet skips auto-sleep", func(t *testing.T) {
		currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		timeNow = func() time.Time { return currentTime }

		pet := newPet(nil)
		pet.Dead = true
		pet.Energy = 0
		pet.Sleeping = false

		applyAutonomousBehavior(&pet)

		if pet.Sleeping {
			t.Error("Dead pet should not auto-sleep")
		}
	})
}

func TestEvolution(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	currentTime := mockTimeNow(t)

	t.Run("Baby to Healthy Child with good care", func(t *testing.T) {
		birthTime := currentTime.Add(-50 * time.Hour) // Just past child threshold

		testCfg := &TestConfig{
			InitialHunger:    90, // Good care
			InitialHappiness: 90,
			InitialEnergy:    90,
			Health:           90,
			LastSavedTime:    birthTime,
		}
		pet := newPet(testCfg)
		saveState(&pet)

		// Fix LastSaved time in file
		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = birthTime
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		if loadedPet.LifeStage != 1 {
			t.Errorf("Expected Child stage (1), got %d", loadedPet.LifeStage)
		}
		if loadedPet.Form != FormHealthyChild {
			t.Errorf("Expected Healthy Child form, got %s", loadedPet.getFormName())
		}
	})

	t.Run("Baby to Troubled Child with poor care", func(t *testing.T) {
		cleanup()
		cleanup = setupTestFile(t)

		birthTime := currentTime.Add(-50 * time.Hour)

		testCfg := &TestConfig{
			InitialHunger:    50, // Poor care (50-69% is poor)
			InitialHappiness: 50,
			InitialEnergy:    50,
			Health:           50,
			LastSavedTime:    birthTime,
		}
		pet := newPet(testCfg)

		// Manually add checkpoints to simulate poor care during baby stage
		for i := 0; i < 48; i++ { // 48 hours of baby stage
			pet.StatCheckpoints["stage_0"] = append(pet.StatCheckpoints["stage_0"], StatCheck{
				Time:      birthTime.Add(time.Duration(i) * time.Hour),
				Hunger:    50,
				Happiness: 50,
				Energy:    50,
				Health:    50,
			})
		}

		saveState(&pet)

		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = birthTime
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		if loadedPet.Form != FormTroubledChild {
			t.Errorf("Expected Troubled Child form, got %s", loadedPet.getFormName())
		}
	})

	t.Run("Healthy Child to Elite Adult with perfect care", func(t *testing.T) {
		cleanup()
		cleanup = setupTestFile(t)

		birthTime := currentTime.Add(-100 * time.Hour) // Past adult threshold

		testCfg := &TestConfig{
			InitialHunger:    95, // Perfect care
			InitialHappiness: 95,
			InitialEnergy:    95,
			Health:           95,
			LastSavedTime:    birthTime,
		}
		pet := newPet(testCfg)
		// Manually set to Healthy Child as if it evolved from baby
		pet.Form = FormHealthyChild
		pet.LifeStage = 1 // Set to child stage
		saveState(&pet)

		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = birthTime
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		if loadedPet.LifeStage != 2 {
			t.Errorf("Expected Adult stage (2), got %d", loadedPet.LifeStage)
		}
		if loadedPet.Form != FormEliteAdult {
			t.Errorf("Expected Elite Adult form, got %s", loadedPet.getFormName())
		}
	})

	t.Run("Baby to Sickly Child with neglect", func(t *testing.T) {
		cleanup()
		cleanup = setupTestFile(t)

		birthTime := currentTime.Add(-50 * time.Hour)

		testCfg := &TestConfig{
			InitialHunger:    15, // Neglect care (<20%)
			InitialHappiness: 15,
			InitialEnergy:    15,
			Health:           15,
			LastSavedTime:    birthTime,
		}
		pet := newPet(testCfg)

		// Manually add checkpoints to simulate neglect during baby stage
		for i := 0; i < 48; i++ {
			pet.StatCheckpoints["stage_0"] = append(pet.StatCheckpoints["stage_0"], StatCheck{
				Time:      birthTime.Add(time.Duration(i) * time.Hour),
				Hunger:    15,
				Happiness: 15,
				Energy:    15,
				Health:    15,
			})
		}

		saveState(&pet)

		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = birthTime
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		if loadedPet.Form != FormSicklyChild {
			t.Errorf("Expected Sickly Child form, got %s", loadedPet.getFormName())
		}
	})

	t.Run("Healthy Child to Standard Adult with good care", func(t *testing.T) {
		cleanup()
		cleanup = setupTestFile(t)

		birthTime := currentTime.Add(-100 * time.Hour)

		testCfg := &TestConfig{
			InitialHunger:    75, // Good care (70-84%)
			InitialHappiness: 75,
			InitialEnergy:    75,
			Health:           75,
			LastSavedTime:    birthTime,
		}
		pet := newPet(testCfg)
		pet.Form = FormHealthyChild
		pet.LifeStage = 1

		// Manually add checkpoints to simulate good care during child stage
		for i := 0; i < 48; i++ {
			pet.StatCheckpoints["stage_1"] = append(pet.StatCheckpoints["stage_1"], StatCheck{
				Time:      birthTime.Add(time.Duration(i) * time.Hour),
				Hunger:    75,
				Happiness: 75,
				Energy:    75,
				Health:    75,
			})
		}

		saveState(&pet)

		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = birthTime
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		if loadedPet.Form != FormStandardAdult {
			t.Errorf("Expected Standard Adult form, got %s", loadedPet.getFormName())
		}
	})

	t.Run("Healthy Child to Grumpy Adult with poor care", func(t *testing.T) {
		cleanup()
		cleanup = setupTestFile(t)

		birthTime := currentTime.Add(-100 * time.Hour)

		testCfg := &TestConfig{
			InitialHunger:    45, // Poor care
			InitialHappiness: 45,
			InitialEnergy:    45,
			Health:           45,
			LastSavedTime:    birthTime,
		}
		pet := newPet(testCfg)
		pet.Form = FormHealthyChild
		pet.LifeStage = 1

		// Manually add checkpoints to simulate poor care during child stage
		for i := 0; i < 48; i++ {
			pet.StatCheckpoints["stage_1"] = append(pet.StatCheckpoints["stage_1"], StatCheck{
				Time:      birthTime.Add(time.Duration(i) * time.Hour),
				Hunger:    45,
				Happiness: 45,
				Energy:    45,
				Health:    45,
			})
		}

		saveState(&pet)

		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = birthTime
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		if loadedPet.Form != FormGrumpyAdult {
			t.Errorf("Expected Grumpy Adult form, got %s", loadedPet.getFormName())
		}
	})

	t.Run("Troubled Child to Redeemed Adult with improved care", func(t *testing.T) {
		cleanup()
		cleanup = setupTestFile(t)

		birthTime := currentTime.Add(-100 * time.Hour)

		testCfg := &TestConfig{
			InitialHunger:    80, // Improved care
			InitialHappiness: 80,
			InitialEnergy:    80,
			Health:           80,
			LastSavedTime:    birthTime,
		}
		pet := newPet(testCfg)
		pet.Form = FormTroubledChild
		pet.LifeStage = 1
		saveState(&pet)

		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = birthTime
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		if loadedPet.Form != FormRedeemedAdult {
			t.Errorf("Expected Redeemed Adult form, got %s", loadedPet.getFormName())
		}
	})

	t.Run("Troubled Child to Delinquent Adult with continued neglect", func(t *testing.T) {
		cleanup()
		cleanup = setupTestFile(t)

		birthTime := currentTime.Add(-100 * time.Hour)

		testCfg := &TestConfig{
			InitialHunger:    30, // Continued poor/neglect care
			InitialHappiness: 30,
			InitialEnergy:    30,
			Health:           30,
			LastSavedTime:    birthTime,
		}
		pet := newPet(testCfg)
		pet.Form = FormTroubledChild
		pet.LifeStage = 1

		// Manually add checkpoints to simulate continued neglect during child stage
		for i := 0; i < 48; i++ {
			pet.StatCheckpoints["stage_1"] = append(pet.StatCheckpoints["stage_1"], StatCheck{
				Time:      birthTime.Add(time.Duration(i) * time.Hour),
				Hunger:    30,
				Happiness: 30,
				Energy:    30,
				Health:    30,
			})
		}

		saveState(&pet)

		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = birthTime
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		if loadedPet.Form != FormDelinquentAdult {
			t.Errorf("Expected Delinquent Adult form, got %s", loadedPet.getFormName())
		}
	})

	t.Run("Sickly Child to Weak Adult", func(t *testing.T) {
		cleanup()
		cleanup = setupTestFile(t)

		birthTime := currentTime.Add(-100 * time.Hour)

		testCfg := &TestConfig{
			InitialHunger:    60, // Any care level - sickly always → weak
			InitialHappiness: 60,
			InitialEnergy:    60,
			Health:           60,
			LastSavedTime:    birthTime,
		}
		pet := newPet(testCfg)
		pet.Form = FormSicklyChild
		pet.LifeStage = 1
		saveState(&pet)

		data, _ := os.ReadFile(testConfigPath)
		var savedPet Pet
		json.Unmarshal(data, &savedPet)
		savedPet.LastSaved = birthTime
		data, _ = json.MarshalIndent(savedPet, "", "  ")
		os.WriteFile(testConfigPath, data, 0644)

		loadedPet := loadState()

		if loadedPet.Form != FormWeakAdult {
			t.Errorf("Expected Weak Adult form, got %s", loadedPet.getFormName())
		}
	})
}
