func TestAgingAndIllness(t *testing.T) {
	cleanup := setupTestFile(t)
	defer cleanup()

	// Create 80-hour old pet with low health
	testCfg := &TestConfig{
		InitialHunger:    maxStat,
		InitialHappiness: maxStat,
		InitialEnergy:    maxStat,
		Health:           40, // Below illness threshold
		Age:              80, // Past natural lifespan
		LastSavedTime:    time.Now().Add(-1 * time.Hour),
	}
	pet := newPet(testCfg)
	saveState(pet)

	// Load and update state
	loadedPet := loadState()

	// Verify natural death
	if !loadedPet.Dead {
		t.Error("Expected natural death from old age")
	}
	if loadedPet.CauseOfDeath != "Old Age" {
		t.Errorf("Expected death cause 'Old Age', got '%s'", loadedPet.CauseOfDeath)
	}

	// Verify illness development
	if !loadedPet.Illness {
		t.Error("Expected pet to develop illness")
	}
}
