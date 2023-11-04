package main

import (
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

func stringExistsInSlice(target string, slice []string) (bool, string) {
	for _, item := range slice {
		if strings.Contains(item, target) {
			remainder := strings.TrimPrefix(item, target)
			return true, strings.TrimSpace(remainder)
		}

	}
	return false, ""
}
func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

func rollAndAdd(input string) (int, error) {
	re := regexp.MustCompile(`(\d*)d(\d+)(\s*([+-])\s*(\d+))?`)
	matches := re.FindStringSubmatch(input)

	if len(matches) == 0 {

		// No dice type foun, try to parse while string as an int
		return strconv.Atoi(strings.TrimSpace(input))

	}

	numberOfDiceStr := matches[1]
	diceType, _ := strconv.Atoi(matches[2])
	modifierStr := matches[5]

	if numberOfDiceStr == "" {
		numberOfDiceStr = "1"
	}
	numberOfDice, _ := strconv.Atoi(numberOfDiceStr)
	modifier, _ := strconv.Atoi(modifierStr)
	return rollDice(numberOfDice, diceType) + modifier, nil
}
func rollDice(numberOfDice, diceType int) int {
	total := 0
	for i := 0; i < numberOfDice; i++ {
		roll := rand.Intn(diceType) + 1
		total = total + roll
	}
	return total
}
func abilityCheck(ability string, list []string) bool {
	if yes, _ := stringExistsInSlice(ability, list); yes {
		return true
	}
	return false
}
