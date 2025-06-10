package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	attackerFiles := []struct {
		name string
		file string
	}{
		{"Captain", "captain.yaml"},
		{"Vindicator", "vindicator.yaml"},
		// Add more attackers here as needed
	}

	defenderFiles := []struct {
		name string
		file string
	}{
		{"Be'lakor", "be'lakor.yaml"},
	}

	for _, att := range attackerFiles {
		var conflict UnitAttackSequence
		conflict.Attacker = loadUnit(att.file)

		for _, def := range defenderFiles {
			fmt.Printf("=== Testing %s against %s ===\n", att.name, def.name)
			conflict.Defender = loadUnit(def.file)

			// Run simulations for statistical analysis
			damages := []int{}

			// Initialize logger only for the first simulation
			initLogger()
			defer closeLogger()

			// Create CSV file for simulation results
			csvFile, err := os.Create(fmt.Sprintf("simulation_results_%s_vs_%s.csv", att.name, def.name))
			if err != nil {
				fmt.Printf("Error creating CSV file: %v\n", err)
				continue
			}
			defer csvFile.Close()

			writer := csv.NewWriter(csvFile)
			defer writer.Flush()

			// Write header
			header := []string{"Simulation", "Total Damage"}
			// Add weapon columns based on first simulation
			damageByLoadout, _ := conflict.loadoutAttackSequence()
			for weaponName := range damageByLoadout {
				header = append(header, weaponName)
			}
			writer.Write(header)

			for i := 0; i < _numSimulations; i++ {
				if i > 0 {
					// Disable detailed combat logging after first simulation
					combatLogger = nil
				}

				damageByLoadout, totalDamage := conflict.loadoutAttackSequence()
				damages = append(damages, totalDamage)

				// Write simulation result to CSV
				row := []string{
					fmt.Sprintf("%d", i+1),
					fmt.Sprintf("%d", totalDamage),
				}
				// Add weapon damage values
				for _, weaponName := range header[2:] {
					row = append(row, fmt.Sprintf("%d", damageByLoadout[weaponName]))
				}
				writer.Write(row)

				// Reload units for next simulation
				conflict.Attacker.Reload()
				conflict.Defender.Reload()
			}

			// Calculate statistics
			sort.Slice(damages, func(i, j int) bool {
				return damages[i] > damages[j]
			})
			mean := float64(0)
			for _, d := range damages {
				mean += float64(d)
			}
			mean /= float64(len(damages))

			fmt.Printf("--- Statistical Analysis (%d simulations) ---\n", _numSimulations)
			fmt.Printf("Mean damage: %.2f\n", mean)
			fmt.Printf("68th percentile: %d\n", damages[int(float64(_numSimulations)*0.32)]) // ~1 standard deviation for normal distribution
			fmt.Printf("95th percentile: %d\n", damages[int(float64(_numSimulations)*0.05)]) // ~2 standard deviations for normal distribution
			fmt.Printf("\n")
		}
	}
}
