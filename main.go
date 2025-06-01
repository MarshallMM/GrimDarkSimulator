package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	attacker := "captain_with_jump_pack.yaml"
	var conflict UnitAttackSequence

	defenderFiles := []struct {
		name string
		file string
	}{
		{"Bladeguard Squad", "bladeguard_veteran_squad.yaml"},
	}

	fmt.Printf("Testing unit loading and loadout attack functionality...\n\n")

	// Test loading the attacker
	fmt.Printf("Loading attacker: %s\n", attacker)
	conflict.Attacker = loadUnit(attacker)
	conflict.Attacker.PrintInfo()
	fmt.Printf("\n")

	for _, def := range defenderFiles {
		fmt.Printf("=== Testing attack against %s (%s) ===\n", def.name, def.file)
		conflict.Defender = loadUnit(def.file)

		// Test a single complete attack sequence (hits + wounds + saves + damage)
		fmt.Printf("\n--- Complete Attack Sequence Test ---\n")
		damage := conflict.loadoutAttackWithWounds()
		fmt.Printf("Total damage applied: %d\n\n", damage)

		// Run simulation for statistics
		x2, y2 := conflict.damageSim()
		fmt.Printf("%s: 68th: %v, 95th: %v\n\n", def.name, x2, y2)
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
		damage = conflict.loadoutAttackWithWounds()
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
