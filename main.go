package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	attacker := "bladeguard_veterans.yaml"
	var conflict UnitAttackSequence

	defenderFiles := []struct {
		name string
		file string
	}{
		{"Bladeguard", "bladeguard_veterans.yaml"},
		// Add more defenders here, e.g. {"Marines", "GenericT4.yaml"}, etc.
	}

	for _, def := range defenderFiles {
		conflict.Defender = loadUnit("/" + def.file)
		conflict.Attacker = loadUnit("/" + attacker)
		x2, y2 := conflict.damageSim()
		fmt.Printf("%s: 68th: %v, 95th: %v\n", def.name, x2, y2)
	}
}

func (conflict *UnitAttackSequence) damageSim() (int, int) {
	var numbers []int
	nSim := 1000

	for i := 0; i < nSim; i++ {
		woundsDealt := 0
		conflict.Defender.Reload()
		conflict.Attacker.Reload()
		// Simulate a single attack sequence here (stub)
		// You should call your actual attack logic here
		// For now, just sum up killed models for percentile calc
		for _, modelDetails := range conflict.Defender.Models {
			woundsDealt += modelDetails.Killed * modelDetails.Wounds
		}
		numbers = append(numbers, woundsDealt)
	}

	sort.Ints(numbers)
	index68thPercentile := int((1 - 0.68) * float64(len(numbers)))
	index95thPercentile := int((1 - 0.95) * float64(len(numbers)))
	value68thPercentile := numbers[index68thPercentile]
	value95thPercentile := numbers[index95thPercentile]

	return int(value68thPercentile), int(value95thPercentile)
}
