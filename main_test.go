package main

import (
	"testing"
	"time"
)

func TestNewPet(t *testing.T) {
	pet := newPet()
	
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
	m := initialModel()
	
	// Test feeding
	originalHunger := m.pet.Hunger
	originalHappiness := m.pet.Happiness
	m.feed()
	
	if m.pet.Hunger <= originalHunger {
		t.Error("Feeding should increase hunger")
	}
	if m.pet.Happiness <= originalHappiness {
		t.Error("Feeding should increase happiness")
	}
	
	// Test playing
	originalEnergy := m.pet.Energy
	originalHappiness = m.pet.Happiness
	m.play()
	
	if m.pet.Energy >= originalEnergy {
		t.Error("Playing should decrease energy")
	}
	if m.pet.Happiness <= originalHappiness {
		t.Error("Playing should increase happiness")
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
	m := initialModel()
	
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
	m := initialModel()
	originalHunger := m.pet.Hunger
	
	// Simulate time passing
	futureTime := time.Now().Add(time.Hour)
	m.updateHourlyStats(futureTime)
	
	if m.pet.Hunger >= originalHunger {
		t.Error("Hunger should decrease over time")
	}
}

func TestGetStatus(t *testing.T) {
	pet := newPet()
	
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
