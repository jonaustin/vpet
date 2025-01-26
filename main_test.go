package main

import (
	"encoding/json"
	"math/rand"
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

	// Create pet that has been critical for 5 hours (new 4h threshold)
	currentTime := time.Now()
	criticalStart := currentTime.Add(-5 * time.Hour)
	testCfg := &TestConfig{
		InitialHunger:    lowStatThreshold - 1,
		InitialHappiness: lowStatThreshold - 1,
		InitialEnergy:    lowStatThreshold - 1,
		Health:           20,  // Force critical health
		LastSavedTime:    criticalStart,
	}
	pet := newPet(testCfg)
	pet.CriticalStartTime = &criticalStart
	pet.CauseOfDeath = "Neglect" // Set expected cause
	saveState(pet)

	// Load state which should trigger death
	loadedPet := loadState()

	if !loadedPet.Dead {
		t.Error("Expected pet to be dead after 4+ hours in critical state")
	}
	if loadedPet.CauseOfDeath != "Neglect" {
		t.Errorf("Expected death cause 'Neglect', got '%s'", loadedPet.CauseOfDeath)
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
		Illness:         true, // Start with illness
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
	currentTime := time.Now()
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
	saveState(pet)

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
		// Create deterministic random source for test
		r := rand.New(rand.NewSource(1))
		randFloat64 = r.Float64 // Override random for test
		
		// Create pet with low health
		testCfg := &TestConfig{
			Health:        40,
			Illness:       false,
			LastSavedTime: time.Now().Add(-1 * time.Hour),
		}
		pet := newPet(testCfg)
		saveState(pet)

		// Load with override of time.Now to match hour elapsed
		loadedPet := func() Pet {
			timeNow = func() time.Time { return testCfg.LastSavedTime.Add(time.Hour) }
			return loadState()
		}()
		if !loadedPet.Illness {
			t.Error("Expected pet to develop illness with low health")
		}
	})

	t.Run("Cure with medicine", func(t *testing.T) {
		// Create sick pet
		testCfg := &TestConfig{
			Health:    40,
			Illness:   true,
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
			Health:    60,
			Illness:   false,
			LastSavedTime: time.Now().Add(-1 * time.Hour),
		}
		pet := newPet(testCfg)
		saveState(pet)

		loadedPet := loadState()
		if loadedPet.Illness {
			t.Error("Pet with health >50 shouldn't develop illness")
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
