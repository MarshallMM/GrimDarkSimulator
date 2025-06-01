package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

const _unitLibraryFilepath = "./library/"
const _heavyComments = true

// Global logger
var combatLogger *zap.Logger

func initLogger() {
	// Configure zap to write to a file
	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Encoding:    "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"combat_log.txt"},
		ErrorOutputPaths: []string{"stderr"},
	}

	var err error
	combatLogger, err = config.Build()
	if err != nil {
		panic(err)
	}
}

func closeLogger() {
	if combatLogger != nil {
		combatLogger.Sync()
	}
}

type UnitAttackSequence struct {
	Attacker Unit
	Defender Unit
}

// New unit structure matching the library builder output
type Unit struct {
	Source         string
	Name           string          `yaml:"name"`
	Type           string          `yaml:"type"`
	Cost           int             `yaml:"cost"`
	Abilities      []string        `yaml:"abilities,omitempty"`
	Keywords       []string        `yaml:"keywords,omitempty"`
	Models         []ModelData     `yaml:"models"`
	LoadoutOptions []LoadoutOption `yaml:"loadout_options,omitempty"`

	// Internal tracking fields
	ModelOrder    []string
	UnitAbilities []string // For legacy compatibility
}

type ModelData struct {
	Name        string                   `yaml:"name"`
	Count       int                      `yaml:"count"`
	Stats       map[string]string        `yaml:"stats,omitempty"`
	BaseLoadout []string                 `yaml:"base_loadout,omitempty"`
	Loadouts    map[string]WeaponProfile `yaml:"loadouts,omitempty"`

	// Internal tracking fields
	Priority        int
	Killed          int
	Wounds          int
	CarryOverWounds int
}

type LoadoutOption struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`
	Options     []string `yaml:"options,omitempty"`
	Description string   `yaml:"description,omitempty"`
}

type WeaponProfile struct {
	Name            string            `yaml:"name"`
	Type            string            `yaml:"type"`
	Characteristics map[string]string `yaml:",inline"`

	// Modifiers for simulation
	Modifiers struct {
		RerollHits    bool `default:"false"`
		RerollHit1s   bool `default:"false"`
		RerollWounds  bool `default:"false"`
		RerollWound1s bool `default:"false"`
		HitMod        int  `default:"0"`
		WoundMod      int  `default:"0"`
		CritHit       int  `default:"6"`
		CritWound     int  `default:"6"`
		CritHitFish   bool `default:"false"`
		CritWoundFish bool `default:"false"`
	}
}

func loadUnit(name string) Unit {
	var (
		data []byte
		err  error
	)

	if data, err = os.ReadFile(_unitLibraryFilepath + name); err != nil {
		panic(err)
	}
	unit := Unit{}
	if err = yaml.Unmarshal(data, &unit); err != nil {
		panic(err)
	}

	// Initialize internal fields
	unit.Source = name
	unit.UnitAbilities = unit.Abilities // Copy abilities for legacy compatibility

	// Process models and initialize tracking fields
	unit.ModelOrder = make([]string, len(unit.Models))
	for i, model := range unit.Models {
		unit.ModelOrder[i] = model.Name

		// Initialize model tracking fields
		unit.Models[i].Priority = i + 1 // Default priority based on order
		unit.Models[i].Killed = 0
		unit.Models[i].CarryOverWounds = 0

		// Parse wounds from stats
		if woundsStr, exists := model.Stats["W"]; exists {
			// Handle different wound formats
			woundsStr = strings.TrimSpace(woundsStr)
			woundsStr = strings.Trim(woundsStr, "\"")

			if wounds, err := strconv.Atoi(woundsStr); err == nil {
				unit.Models[i].Wounds = wounds
			} else {
				// Default to 1 wound if parsing fails
				unit.Models[i].Wounds = 1
			}
		} else {
			// Default to 1 wound if no W stat found
			unit.Models[i].Wounds = 1
		}

		// Initialize weapon modifiers
		if unit.Models[i].Loadouts != nil {
			for weaponName, weapon := range unit.Models[i].Loadouts {
				weapon.Modifiers.CritHit = 6
				weapon.Modifiers.CritHitFish = false
				weapon.Modifiers.CritWound = 6
				weapon.Modifiers.CritWoundFish = false
				weapon.Modifiers.HitMod = 0
				weapon.Modifiers.WoundMod = 0
				weapon.Modifiers.RerollHit1s = false
				weapon.Modifiers.RerollHits = false
				weapon.Modifiers.RerollWound1s = false
				weapon.Modifiers.RerollWounds = false
				unit.Models[i].Loadouts[weaponName] = weapon
			}
		}
	}

	return unit
}

func (u *Unit) PrintInfo() {
	if _heavyComments {
		fmt.Printf("Unit: %s | Cost: %d | Type: %s\n", u.Name, u.Cost, u.Type)
		fmt.Printf("Abilities: %v\n", u.Abilities)
		fmt.Printf("Keywords: %v\n", u.Keywords)

		for i, model := range u.Models {
			fmt.Printf("Model[%d]: %s | Count: %d | Killed: %d | Wounds: %d\n",
				i, model.Name, model.Count, model.Killed, model.CarryOverWounds)
			fmt.Printf("  Stats: %v\n", model.Stats)

			if model.Loadouts != nil {
				for weaponName, weapon := range model.Loadouts {
					fmt.Printf("  Weapon: %s | Type: %s | Characteristics: %v\n",
						weaponName, weapon.Type, weapon.Characteristics)
				}
			}
		}
	}
}

func (u *Unit) calcToughness() int {
	toughness := 0
	// Iterate through models and get highest toughness
	for _, model := range u.Models {
		if model.Count > model.Killed {
			// Check if model is not a leader (for bodyguard units)
			isLeader := false
			for _, ability := range u.Abilities {
				if strings.Contains(strings.ToLower(ability), "leader") {
					isLeader = true
					break
				}
			}

			if !isLeader {
				if tStr, exists := model.Stats["T"]; exists {
					if t, err := strconv.Atoi(tStr); err == nil && toughness < t {
						toughness = t
					}
				}
			}
		}
	}
	return toughness
}

func (u *Unit) removeAbility(ability2remove string) bool {
	used := false
	for i, ability := range u.UnitAbilities {
		if ability == ability2remove {
			u.UnitAbilities = remove(u.UnitAbilities, i)
			used = true
			break
		}
	}
	// Also remove from main abilities list
	for i, ability := range u.Abilities {
		if ability == ability2remove {
			u.Abilities = remove(u.Abilities, i)
			break
		}
	}
	return used
}

func (conflict *UnitAttackSequence) applyDamage(modelIndex int, damString string, params ...string) {
	var (
		err     error
		damage  int
		mortals bool
	)
	if exists, _ := stringExistsInSlice("mortal", params); exists {
		mortals = true
	}

	if damage, err = rollAndAdd(damString); err != nil {
		fmt.Println(err)
	}

	if modelIndex >= len(conflict.Defender.Models) {
		return
	}

	model := &conflict.Defender.Models[modelIndex]

	// Parse FNP values from abilities or characteristics
	fnp := 0
	mwfnp := 0

	// Check unit abilities for FNP
	for _, ability := range conflict.Defender.Abilities {
		if strings.Contains(strings.ToLower(ability), "feel no pain") {
			// Try to extract FNP value - this would need more sophisticated parsing
			// For now, assume common values
			if strings.Contains(ability, "5+") {
				fnp = 5
			} else if strings.Contains(ability, "6+") {
				fnp = 6
			} else if strings.Contains(ability, "4+") {
				fnp = 4
			}
		}
	}

	if fnp > 0 || (mortals && mwfnp > 0) {
		threshold := 0
		if fnp > threshold {
			threshold = fnp
		}
		if mortals && mwfnp > threshold {
			threshold = mwfnp
		}

		initialDamage := damage
		for i := 0; i < initialDamage; i++ {
			roll := rollDice(1, 6)
			if roll >= threshold {
				damage = damage - 1
			}
		}
	}

	// Check for special abilities like NECRODERMIS
	if abilityCheck("NECRODERMIS", conflict.Defender.UnitAbilities) {
		damage = int(math.Ceil(float64(damage) / 2))
	}

	model.CarryOverWounds = model.CarryOverWounds + damage
	if model.CarryOverWounds >= model.Wounds && model.Killed < model.Count {
		model.Killed++
		model.CarryOverWounds = 0
	}

	// Update the model in the slice
	conflict.Defender.Models[modelIndex] = *model
}

func (u *Unit) Reload() {
	reloadUnit := loadUnit(u.Source)
	for i := range u.Models {
		u.Models[i].Killed = 0
		u.Models[i].CarryOverWounds = 0
		// Restore original counts and stats
		u.Models[i].Count = reloadUnit.Models[i].Count
		u.Models[i].Wounds = reloadUnit.Models[i].Wounds
	}
}

func (u *Unit) Reset() {
	for i := range u.Models {
		u.Models[i].Killed = 0
		u.Models[i].CarryOverWounds = 0
	}
}

// Helper function to get model by name (for legacy compatibility)
func (u *Unit) GetModelByName(name string) *ModelData {
	for i := range u.Models {
		if u.Models[i].Name == name {
			return &u.Models[i]
		}
	}
	return nil
}

// Helper function to get all weapons from all models
func (u *Unit) GetAllWeapons() map[string]WeaponProfile {
	allWeapons := make(map[string]WeaponProfile)
	for _, model := range u.Models {
		if model.Loadouts != nil {
			for weaponName, weapon := range model.Loadouts {
				allWeapons[weaponName] = weapon
			}
		}
	}
	return allWeapons
}

// Helper function to convert characteristic strings to integers where needed
func (w *WeaponProfile) GetIntCharacteristic(name string) int {
	if value, exists := w.Characteristics[name]; exists {
		// Handle different formats
		value = strings.TrimSpace(value)

		// Remove quotes if present
		value = strings.Trim(value, "\"")

		// Handle dice notation (just return 0 for now, could be enhanced)
		if strings.Contains(value, "D") || value == "N/A" || value == "-" {
			return 0
		}

		// Handle ranges like "12\""
		if strings.Contains(value, "\"") {
			value = strings.Replace(value, "\"", "", -1)
		}

		// Handle skill values like "3+"
		if strings.Contains(value, "+") {
			value = strings.Replace(value, "+", "", -1)
		}

		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return 0
}

// Helper function to get string characteristic
func (w *WeaponProfile) GetStringCharacteristic(name string) string {
	if value, exists := w.Characteristics[name]; exists {
		return value
	}
	return ""
}

// Enhanced loadout attack method that includes wound rolling and saves
func (conflict *UnitAttackSequence) loadoutAttackWithWounds() int {
	totalDamage := 0

	// Find the first alive model in the defender for targeting
	var targetModel *ModelData
	targetModelIndex := -1
	for i, model := range conflict.Defender.Models {
		if model.Killed < model.Count {
			targetModel = &conflict.Defender.Models[i]
			targetModelIndex = i
			break
		}
	}

	if targetModel == nil {
		fmt.Printf("No alive models to target\n")
		return 0
	}

	fmt.Printf("Targeting: %s (T%s, SV%s", targetModel.Name, targetModel.Stats["T"], targetModel.Stats["SV"])
	if isvStr, hasISV := targetModel.Stats["ISV"]; hasISV {
		fmt.Printf(", ISV%s", isvStr)
	}
	fmt.Printf(")\n")

	// Iterate through all models in the attacker
	for modelIndex, model := range conflict.Attacker.Models {
		// Skip killed models
		if model.Killed >= model.Count {
			continue
		}

		// Calculate how many of this model are still alive
		aliveCount := model.Count - model.Killed

		// Iterate through each loadout/weapon for this model
		if model.Loadouts != nil {
			for weaponName, weapon := range model.Loadouts {
				fmt.Printf("Model %d (%s) attacking with %s\n", modelIndex, model.Name, weaponName)

				// Log weapon attack start
				if combatLogger != nil {
					combatLogger.Info("Starting weapon attack",
						zap.String("attacker_model", model.Name),
						zap.String("weapon_name", weaponName),
						zap.String("weapon_type", weapon.Type),
						zap.String("target_model", targetModel.Name))
				}

				// Get number of attacks
				attacksStr := weapon.GetStringCharacteristic("A")
				attacks := 0

				// Handle different attack formats
				if attacksStr != "" {
					if attacksInt, err := strconv.Atoi(attacksStr); err == nil {
						attacks = attacksInt
					} else {
						// Handle dice notation like "D6", "2D6", etc.
						diceStr := strings.ToLower(attacksStr)
						if strings.HasPrefix(diceStr, "d") {
							diceStr = "1" + diceStr
						}
						if attacksDice, err := rollAndAdd(diceStr); err == nil {
							attacks = attacksDice
						} else {
							fmt.Printf("  Could not parse attacks '%s', skipping\n", attacksStr)
							continue
						}
					}
				} else {
					fmt.Printf("  No attacks found for weapon, skipping\n")
					continue
				}

				// Calculate total attacks (attacks per weapon * number of alive models)
				totalAttacks := attacks * aliveCount

				// PHASE 1: Roll for hits
				hits := 0
				sustainedHits := 0
				lethalHits := 0

				// Check if weapon has Torrent (auto-hit)
				keywords := weapon.GetStringCharacteristic("Keywords")
				isTorrent := strings.Contains(strings.ToLower(keywords), "torrent")

				if isTorrent {
					// Torrent weapons auto-hit
					fmt.Printf("  %d attacks, auto-hit (Torrent)\n", totalAttacks)
					hits = totalAttacks

					if combatLogger != nil {
						combatLogger.Info("Hit Phase - Torrent",
							zap.Int("total_attacks", totalAttacks),
							zap.Bool("auto_hit", true),
							zap.Int("total_hits", hits))
					}
				} else {
					// Get skill value (BS for ranged, WS for melee)
					var skillStr string
					if strings.Contains(strings.ToLower(weapon.Type), "ranged") {
						skillStr = weapon.GetStringCharacteristic("BS")
					} else if strings.Contains(strings.ToLower(weapon.Type), "melee") {
						skillStr = weapon.GetStringCharacteristic("WS")
					} else {
						fmt.Printf("  Unknown weapon type '%s', assuming ranged\n", weapon.Type)
						skillStr = weapon.GetStringCharacteristic("BS")
					}

					// Convert skill from "3+" to 3
					skillValue := 0
					if skillStr != "" {
						skillStr = strings.TrimSpace(skillStr)
						skillStr = strings.Replace(skillStr, "+", "", -1)
						if skill, err := strconv.Atoi(skillStr); err == nil {
							skillValue = skill
						} else {
							fmt.Printf("  Could not parse skill '%s', skipping\n", skillStr)
							continue
						}
					} else {
						fmt.Printf("  No skill value found for weapon, skipping\n")
						continue
					}

					// Add hit modifier
					finalSkill := skillValue - weapon.Modifiers.HitMod
					if finalSkill < 2 {
						finalSkill = 2 // Minimum hit on 2+
					}
					if finalSkill > 6 {
						finalSkill = 6 // Maximum hit on 6+
					}

					fmt.Printf("  %d attacks, need %d+ to hit (base %d+ with %+d modifier)\n",
						totalAttacks, finalSkill, skillValue, weapon.Modifiers.HitMod)

					if combatLogger != nil {
						combatLogger.Info("Hit Phase - Rolling",
							zap.Int("total_attacks", totalAttacks),
							zap.Int("hit_threshold", finalSkill),
							zap.Int("base_skill", skillValue),
							zap.Int("hit_modifier", weapon.Modifiers.HitMod))
					}

					// Roll for hits individually
					for i := 0; i < totalAttacks; i++ {
						roll := rollDice(1, 6)
						hit := roll >= finalSkill
						criticalHit := roll >= weapon.Modifiers.CritHit
						rerolled := false

						// Handle rerolls for misses
						if !hit {
							shouldReroll := false
							if weapon.Modifiers.RerollHits {
								shouldReroll = true
							} else if weapon.Modifiers.RerollHit1s && roll == 1 {
								shouldReroll = true
							}

							if shouldReroll {
								rerollResult := rollDice(1, 6)
								hit = rerollResult >= finalSkill
								criticalHit = rerollResult >= weapon.Modifiers.CritHit
								rerolled = true

								if combatLogger != nil {
									combatLogger.Info("Hit Reroll",
										zap.Int("attack_number", i+1),
										zap.Int("original_roll", roll),
										zap.Int("reroll", rerollResult),
										zap.Int("threshold", finalSkill),
										zap.Bool("hit_after_reroll", hit),
										zap.Bool("critical_after_reroll", criticalHit))
								}

								roll = rerollResult // Update roll for logging
							}
						}

						if hit {
							hits++

							// Handle critical hit effects
							if criticalHit {
								// Check for Lethal Hits
								if strings.Contains(strings.ToLower(keywords), "lethal hits") {
									lethalHits++
								}

								// Check for Sustained Hits
								if strings.Contains(strings.ToLower(keywords), "sustained hits") {
									// Parse sustained hits value - look for "sustained hits X"
									keywordParts := strings.Fields(strings.ToLower(keywords))
									for j, part := range keywordParts {
										if part == "sustained" && j+2 < len(keywordParts) && keywordParts[j+1] == "hits" {
											if sustainedValue, err := strconv.Atoi(keywordParts[j+2]); err == nil {
												sustainedHits += sustainedValue
											}
											break
										}
									}
								}
							}
						}

						if combatLogger != nil {
							combatLogger.Info("Hit Roll",
								zap.Int("attack_number", i+1),
								zap.Int("roll", roll),
								zap.Int("threshold", finalSkill),
								zap.Bool("hit", hit),
								zap.Bool("critical_hit", criticalHit),
								zap.Bool("rerolled", rerolled),
								zap.Int("running_hits", hits))
						}
					}

					// Add sustained hits to total
					hits += sustainedHits

					fmt.Printf("  Scored %d hits out of %d attacks", hits, totalAttacks)
					if sustainedHits > 0 {
						fmt.Printf(" (including %d Sustained Hits)", sustainedHits)
					}
					if lethalHits > 0 {
						fmt.Printf(" (%d Lethal Hits auto-wound)", lethalHits)
					}
					fmt.Printf("\n")

					if combatLogger != nil {
						combatLogger.Info("Hit Phase Complete",
							zap.Int("total_hits", hits),
							zap.Int("sustained_hits", sustainedHits),
							zap.Int("lethal_hits", lethalHits))
					}
				}

				// PHASE 2: Roll for wounds
				wounds, criticalWounds := conflict.rollWoundsWithDetailedLogging(hits, weapon, targetModel, lethalHits)

				// PHASE 3: Roll for saves and apply damage
				damageApplied := 0
				if wounds > 0 {
					damageApplied = conflict.rollSavesWithDetailedLogging(wounds, criticalWounds, weapon, targetModel, targetModelIndex)
					totalDamage += damageApplied
				}

				// Log remaining defenders after damage
				if combatLogger != nil {
					remainingModels := 0
					totalWoundsRemaining := 0
					for _, defModel := range conflict.Defender.Models {
						alive := defModel.Count - defModel.Killed
						if alive > 0 {
							remainingModels += alive
							totalWoundsRemaining += alive * defModel.Wounds
						}
					}

					combatLogger.Info("Weapon Attack Complete",
						zap.Int("damage_applied", damageApplied),
						zap.Int("remaining_models", remainingModels),
						zap.Int("remaining_wounds", totalWoundsRemaining))
				}

				fmt.Printf("\n")
			}
		}
	}

	fmt.Printf("Total damage applied: %d\n", totalDamage)
	if combatLogger != nil {
		combatLogger.Info("Combat Complete", zap.Int("total_damage", totalDamage))
	}
	return totalDamage
}

// Wound rolling method with detailed logging for each roll
func (conflict *UnitAttackSequence) rollWoundsWithDetailedLogging(hits int, weapon WeaponProfile, targetModel *ModelData, lethalHits int) (int, int) {
	if hits <= 0 {
		return 0, 0
	}

	// Get weapon strength
	strengthStr := weapon.GetStringCharacteristic("S")
	strength := 0
	if strengthStr != "" {
		if s, err := strconv.Atoi(strengthStr); err == nil {
			strength = s
		} else {
			fmt.Printf("  Could not parse weapon strength '%s'\n", strengthStr)
			return 0, 0
		}
	} else {
		fmt.Printf("  No strength found for weapon\n")
		return 0, 0
	}

	// Get target toughness
	toughnessStr := targetModel.Stats["T"]
	toughness := 0
	if toughnessStr != "" {
		if t, err := strconv.Atoi(toughnessStr); err == nil {
			toughness = t
		} else {
			fmt.Printf("  Could not parse target toughness '%s'\n", toughnessStr)
			return 0, 0
		}
	} else {
		fmt.Printf("  No toughness found for target\n")
		return 0, 0
	}

	// Calculate wound threshold based on Strength vs Toughness
	var woundThreshold int
	if strength == toughness {
		woundThreshold = 4 // S = T: Need 4+
	} else if strength >= 2*toughness {
		woundThreshold = 2 // S >= 2*T: Need 2+
	} else if strength > toughness {
		woundThreshold = 3 // S > T: Need 3+
	} else if strength*2 <= toughness {
		woundThreshold = 6 // S*2 <= T: Need 6+
	} else { // strength < toughness
		woundThreshold = 5 // S < T: Need 5+
	}

	// Apply wound modifier
	finalWoundThreshold := woundThreshold - weapon.Modifiers.WoundMod
	if finalWoundThreshold < 2 {
		finalWoundThreshold = 2 // Minimum wound on 2+
	}
	if finalWoundThreshold > 6 {
		finalWoundThreshold = 6 // Maximum wound on 6+
	}

	fmt.Printf("  Rolling %d hits to wound: S%d vs T%d, need %d+ (base %d+ with %+d modifier)",
		hits, strength, toughness, finalWoundThreshold, woundThreshold, weapon.Modifiers.WoundMod)
	if lethalHits > 0 {
		fmt.Printf(" (%d Lethal Hits auto-wound)", lethalHits)
	}
	fmt.Printf("\n")

	if combatLogger != nil {
		combatLogger.Info("Wound Phase - Starting",
			zap.Int("hits", hits),
			zap.Int("lethal_hits", lethalHits),
			zap.Int("weapon_strength", strength),
			zap.Int("target_toughness", toughness),
			zap.Int("wound_threshold", finalWoundThreshold),
			zap.Int("base_threshold", woundThreshold),
			zap.Int("wound_modifier", weapon.Modifiers.WoundMod))
	}

	// Start with lethal hits that auto-wound
	wounds := lethalHits
	criticalWounds := 0

	// Process remaining hits that need wound rolls
	normalHits := hits - lethalHits
	if normalHits < 0 {
		normalHits = 0
	}

	// Roll for wounds individually for non-lethal hits
	for i := 0; i < normalHits; i++ {
		roll := rollDice(1, 6)
		wound := roll >= finalWoundThreshold
		criticalWound := roll >= weapon.Modifiers.CritWound
		rerolled := false

		// Handle rerolls for misses
		if !wound {
			shouldReroll := false
			if weapon.Modifiers.RerollWounds {
				shouldReroll = true
			} else if weapon.Modifiers.RerollWound1s && roll == 1 {
				shouldReroll = true
			}

			if shouldReroll {
				rerollResult := rollDice(1, 6)
				wound = rerollResult >= finalWoundThreshold
				criticalWound = rerollResult >= weapon.Modifiers.CritWound
				rerolled = true

				if combatLogger != nil {
					combatLogger.Info("Wound Reroll",
						zap.Int("hit_number", i+1),
						zap.Int("original_roll", roll),
						zap.Int("reroll", rerollResult),
						zap.Int("threshold", finalWoundThreshold),
						zap.Bool("wound_after_reroll", wound),
						zap.Bool("devastating_wound_after_reroll", criticalWound))
				}

				roll = rerollResult // Update roll for logging
			}
		}

		if wound {
			wounds++
			if criticalWound {
				criticalWounds++
			}
		}

		if combatLogger != nil {
			combatLogger.Info("Wound Roll",
				zap.Int("hit_number", i+1),
				zap.Int("roll", roll),
				zap.Int("threshold", finalWoundThreshold),
				zap.Bool("wound", wound),
				zap.Bool("devastating_wound", criticalWound),
				zap.Bool("rerolled", rerolled),
				zap.Int("running_wounds", wounds))
		}
	}

	fmt.Printf("  Scored %d wounds out of %d hits", wounds, hits)
	if criticalWounds > 0 {
		fmt.Printf(" (%d Devastating Wounds bypass saves)", criticalWounds)
	}
	fmt.Printf("\n")

	if combatLogger != nil {
		combatLogger.Info("Wound Phase Complete",
			zap.Int("total_wounds", wounds),
			zap.Int("devastating_wounds", criticalWounds),
			zap.Int("lethal_hits_autowound", lethalHits))
	}

	return wounds, criticalWounds
}

// Save rolling method with detailed logging for each roll
func (conflict *UnitAttackSequence) rollSavesWithDetailedLogging(wounds int, criticalWounds int, weapon WeaponProfile, targetModel *ModelData, targetModelIndex int) int {
	if wounds <= 0 {
		return 0
	}

	// Get weapon AP and Damage
	apStr := weapon.GetStringCharacteristic("AP")
	ap := 0
	if apStr != "" && apStr != "-" {
		// AP is usually negative (like "-1", "-2"), but stored as positive in some systems
		apStr = strings.TrimSpace(apStr)
		if strings.HasPrefix(apStr, "-") {
			apStr = strings.TrimPrefix(apStr, "-")
		}
		if apVal, err := strconv.Atoi(apStr); err == nil {
			ap = apVal // Store as positive value
		}
	}

	damageStr := weapon.GetStringCharacteristic("D")
	if damageStr == "" {
		damageStr = "1" // Default to 1 damage
	}

	// Get target saves
	svStr := targetModel.Stats["SV"]
	sv := 7 // Default to no save (7+ is impossible)
	if svStr != "" {
		svStr = strings.TrimSpace(svStr)
		svStr = strings.Replace(svStr, "+", "", -1)
		if svVal, err := strconv.Atoi(svStr); err == nil {
			sv = svVal
		}
	}

	isvStr := targetModel.Stats["ISV"]
	isv := 7 // Default to no invuln save
	if isvStr != "" {
		isvStr = strings.TrimSpace(isvStr)
		isvStr = strings.Replace(isvStr, "+", "", -1)
		if isvVal, err := strconv.Atoi(isvStr); err == nil {
			isv = isvVal
		}
	}

	fmt.Printf("  Rolling %d wounds for saves: SV %d+ (modified by AP-%d), ISV %d+",
		wounds, sv, ap, isv)
	if criticalWounds > 0 {
		fmt.Printf(" (%d Devastating Wounds bypass saves)", criticalWounds)
	}
	fmt.Printf("\n")

	if combatLogger != nil {
		combatLogger.Info("Save Phase - Starting",
			zap.Int("wounds", wounds),
			zap.Int("devastating_wounds", criticalWounds),
			zap.Int("armor_save", sv),
			zap.Int("invulnerable_save", isv),
			zap.Int("weapon_ap", ap),
			zap.String("weapon_damage", damageStr))
	}

	savedWounds := 0
	damageApplied := 0

	// Process critical wounds first (they bypass saves)
	for i := 0; i < criticalWounds; i++ {
		if combatLogger != nil {
			combatLogger.Info("Devastating Wound",
				zap.Int("wound_number", i+1),
				zap.Bool("bypasses_save", true))
		}

		// Apply damage for Devastating Wound (no save allowed)
		conflict.applyDamage(targetModelIndex, damageStr)
		damageApplied++

		if combatLogger != nil {
			combatLogger.Info("Damage Applied",
				zap.String("damage_amount", damageStr),
				zap.Int("target_model_index", targetModelIndex),
				zap.String("target_model_name", targetModel.Name),
				zap.Bool("devastating_wound", true))
		}
	}

	// Process remaining wounds that allow saves
	normalWounds := wounds - criticalWounds
	if normalWounds < 0 {
		normalWounds = 0
	}

	for i := 0; i < normalWounds; i++ {
		roll := rollDice(1, 6)
		saved := false
		saveType := ""
		saveUsed := 0

		// Check invulnerable save first (not modified by AP)
		if isv <= 6 && roll >= isv {
			saved = true
			saveType = "invulnerable"
			saveUsed = isv
		} else {
			// Check armor save (modified by AP)
			modifiedSv := sv + ap
			if modifiedSv <= 6 && roll >= modifiedSv {
				saved = true
				saveType = "armor"
				saveUsed = modifiedSv
			}
		}

		if saved {
			savedWounds++
			if _heavyComments {
				fmt.Printf("    Wound %d: Rolled %d, saved (%s save %d+)\n", i+1, roll, saveType, saveUsed)
			}

			if combatLogger != nil {
				combatLogger.Info("Save Roll",
					zap.Int("wound_number", i+1+criticalWounds),
					zap.Int("roll", roll),
					zap.String("save_type", saveType),
					zap.Int("save_threshold", saveUsed),
					zap.Bool("saved", true))
			}
		} else {
			if _heavyComments {
				fmt.Printf("    Wound %d: Rolled %d, failed save\n", i+1, roll)
			}

			if combatLogger != nil {
				combatLogger.Info("Save Roll",
					zap.Int("wound_number", i+1+criticalWounds),
					zap.Int("roll", roll),
					zap.String("save_type", "failed"),
					zap.Int("save_threshold", 0),
					zap.Bool("saved", false))
			}

			// Apply damage for failed save
			conflict.applyDamage(targetModelIndex, damageStr)
			damageApplied++

			if combatLogger != nil {
				combatLogger.Info("Damage Applied",
					zap.String("damage_amount", damageStr),
					zap.Int("target_model_index", targetModelIndex),
					zap.String("target_model_name", targetModel.Name),
					zap.Bool("devastating_wound", false))
			}
		}
	}

	fmt.Printf("  Saves: %d saved, %d damage applied (%d from Devastating Wounds, %d from failed saves)\n",
		savedWounds, damageApplied, criticalWounds, damageApplied-criticalWounds)

	if combatLogger != nil {
		combatLogger.Info("Save Phase Complete",
			zap.Int("total_damage_applied", damageApplied),
			zap.Int("devastating_wound_damage", criticalWounds),
			zap.Int("failed_save_damage", damageApplied-criticalWounds),
			zap.Int("wounds_saved", savedWounds))
	}

	return damageApplied
}
