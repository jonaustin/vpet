package main

import (
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
		os.RemoveAll(tmpDir)
	}
}

func TestNewPet(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()
	pet := newPet(nil)

	if pet.Name != defaultPetName {
		t.Errorf("Expected pet name to be %s, got %s", defaultPetName, pet.Name)
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
		IsSleeping:       false,
		LastSavedTime:    time.Now(),
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
	currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return currentTime }

	// Create initial pet state from 2 hours ago
	twoHoursAgo := currentTime.Add(-2 * time.Hour)
	testCfg := &TestConfig{
		InitialHunger:    maxStat,      // Start at 100%
		InitialHappiness: maxStat,      // Start at 100%
		InitialEnergy:    maxStat,      // Start at 100%
		IsSleeping:       false,
		LastSavedTime:    twoHoursAgo,  // Set last saved to 2 hours ago
	}

	// Save initial state
	pet := newPet(testCfg)
	saveState(pet)

	// Load state which will process the elapsed time
	loadedPet := loadState()

	// Verify stats decreased appropriately for 2 hours
	expectedHunger := maxStat - (2 * hungerDecreaseRate)  // 2 hours * 5 per hour = 10 decrease
	if loadedPet.Hunger != expectedHunger {
		t.Errorf("Expected hunger to be %d after 2 hours, got %d", expectedHunger, loadedPet.Hunger)
	}

	expectedEnergy := maxStat - energyDecreaseRate  // 2 hours = 1 energy decrease
	if loadedPet.Energy != expectedEnergy {
		t.Errorf("Expected energy to be %d after 2 hours, got %d", expectedEnergy, loadedPet.Energy)
	}

	// Happiness shouldn't decrease since hunger and energy are still above threshold
	if loadedPet.Happiness != maxStat {
		t.Errorf("Expected happiness to stay at %d, got %d", maxStat, loadedPet.Happiness)
	}
}

func TestGetStatus(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()
	pet := newPet(nil)

	// Test sleeping status
	pet.Sleeping = true
	if status := getStatus(pet); status != "😴 Sleeping" {
		t.Errorf("Expected sleeping status, got %s", status)
	}

	// Test hungry status
	pet.Sleeping = false
	pet.Hunger = lowStatThreshold - 1
	if status := getStatus(pet); status != "🙀 Hungry" {
		t.Errorf("Expected hungry status, got %s", status)
	}

	// Test happy status
	pet.Hunger = maxStat
	pet.Energy = maxStat
	pet.Happiness = maxStat
	if status := getStatus(pet); status != "😸 Happy" {
		t.Errorf("Expected happy status, got %s", status)
	}
}
