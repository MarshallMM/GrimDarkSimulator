package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	attacker := "test_heavy_squad.yaml"
	var conflict UnitAttackSequence

	defenderFiles := []struct {
		name string
		file string
	}{
		{"Bladeguard Squad", "bladeguard_veteran_squad.yaml"},
	}

	fmt.Printf("Testing abilities system with Test Heavy Squad...\n\n")

	// Test loading the attacker
	fmt.Printf("Loading attacker: %s\n", attacker)
	conflict.Attacker = loadUnit(attacker)
	conflict.Attacker.PrintInfo()
	fmt.Printf("\n")

	for _, def := range defenderFiles {
		fmt.Printf("=== Testing attack against %s (%s) ===\n", def.name, def.file)
		conflict.Defender = loadUnit(def.file)

		// Test a single complete attack sequence (hits + wounds + saves + damage)
		fmt.Printf("\n--- Single Attack Test ---\n")
		damage := conflict.loadoutAttackSequence()
		fmt.Printf("Total damage dealt: %d\n", damage)

		// Reload units for clean state
		conflict.Attacker.Reload()
		conflict.Defender.Reload()

		// Run 100 simulations for statistical analysis
		damages := []int{}

		// Initialize logger only for the first simulation
		initLogger()
		defer closeLogger()

		for i := 0; i < 100; i++ {
			if i > 0 {
				// Disable logging after first simulation
				combatLogger = nil
			}

			damage = conflict.loadoutAttackSequence()
			damages = append(damages, damage)

			// Reload units for next simulation
			conflict.Attacker.Reload()
			conflict.Defender.Reload()
		}

		// Calculate statistics
		sort.Ints(damages)
		mean := float64(0)
		for _, d := range damages {
			mean += float64(d)
		}
		mean /= float64(len(damages))

		fmt.Printf("\n--- Statistical Analysis (100 simulations) ---\n")
		fmt.Printf("Mean damage: %.2f\n", mean)
		fmt.Printf("Min damage: %d\n", damages[0])
		fmt.Printf("Max damage: %d\n", damages[len(damages)-1])
		fmt.Printf("68th percentile: %d\n", damages[67]) // ~1 standard deviation for normal distribution
		fmt.Printf("95th percentile: %d\n", damages[94]) // ~2 standard deviations for normal distribution
		fmt.Printf("Range of outcomes: %d-%d damage\n", damages[0], damages[len(damages)-1])
	}
}

func (conflict *UnitAttackSequence) damageSim() (int, int) {
	var numbers []int
	nSim := 100 // Reduced for initial testing

	for i := 0; i < nSim; i++ {
		// Initialize logger only for the first run
		if i == 0 {
			initLogger()
			defer closeLogger()
		}

		damage := 0
		conflict.Defender.Reload()
		conflict.Attacker.Reload()

		// Use the complete attack sequence with saves and damage
		damage = conflict.loadoutAttackSequence()
		numbers = append(numbers, damage)

		// Disable logger after first run
		if i == 0 {
			combatLogger.Sync()
			combatLogger = nil
		}
	}

	sort.Ints(numbers)
	index68thPercentile := int((1 - 0.68) * float64(len(numbers)))
	index95thPercentile := int((1 - 0.95) * float64(len(numbers)))
	value68thPercentile := numbers[index68thPercentile]
	value95thPercentile := numbers[index95thPercentile]

	return int(value68thPercentile), int(value95thPercentile)
}
