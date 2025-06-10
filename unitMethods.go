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
const _numSimulations = 1000

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

func (conflict *UnitAttackSequence) applyDamage(modelIndex int, damString string, params ...string) int {
	var (
		err         error
		damage      int
		mortals     bool
		devastating bool
	)
	if exists, _ := stringExistsInSlice("mortal", params); exists {
		mortals = true
	}
	if exists, _ := stringExistsInSlice("devastating", params); exists {
		devastating = true
	}

	// Handle damage string parsing
	damString = strings.TrimSpace(damString)
	if damString == "" {
		damString = "1" // Default to 1 damage if empty
	}

	// Handle dice notation
	if strings.Contains(strings.ToLower(damString), "d") {
		// Handle D6, 2D6, etc.
		diceStr := strings.ToLower(damString)
		if strings.HasPrefix(diceStr, "d") {
			diceStr = "1" + diceStr // Convert "D6" to "1D6"
		}
		if damage, err = rollAndAdd(diceStr); err != nil {
			fmt.Printf("Error rolling damage dice '%s': %v\n", diceStr, err)
			damage = 1 // Default to 1 damage on error
		}
	} else {
		// Handle numeric damage
		if damage, err = strconv.Atoi(damString); err != nil {
			fmt.Printf("Error parsing damage '%s': %v\n", damString, err)
			damage = 1 // Default to 1 damage on error
		}
	}

	if modelIndex >= len(conflict.Defender.Models) {
		return 0
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

	// Calculate remaining health before applying damage
	remainingHealth := model.Wounds - model.CarryOverWounds
	aliveModels := model.Count - model.Killed

	model.CarryOverWounds = model.CarryOverWounds + damage
	if model.CarryOverWounds >= model.Wounds && model.Killed < model.Count {
		model.Killed++
		model.CarryOverWounds = 0
	}

	// Calculate new remaining health
	newRemainingHealth := model.Wounds - model.CarryOverWounds
	newAliveModels := model.Count - model.Killed

	// Log health changes
	if combatLogger != nil {
		combatLogger.Info("Damage Applied",
			zap.Int("damage_amount", damage),
			zap.String("damage_characteristic", damString),
			zap.Int("target_model_index", modelIndex),
			zap.String("target_model_name", model.Name),
			zap.Int("previous_health", remainingHealth),
			zap.Int("new_health", newRemainingHealth),
			zap.Int("previous_alive_models", aliveModels),
			zap.Int("new_alive_models", newAliveModels),
			zap.Int("total_wounds", model.Wounds),
			zap.Int("total_models", model.Count),
			zap.Bool("devastating_wound", devastating),
			zap.Bool("mortal_wound", mortals))
	}

	// Update the model in the slice
	conflict.Defender.Models[modelIndex] = *model

	return damage
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
func (conflict *UnitAttackSequence) loadoutAttackSequence() (map[string]int, int) {
	// Apply abilities and weapon modifications at start of combat
	conflict.applyAbilities()

	totalDamage := 0
	damageByLoadout := make(map[string]int)

	// Initialize damage tracking for all weapons to 0
	for _, model := range conflict.Attacker.Models {
		if model.Loadouts != nil {
			for weaponName := range model.Loadouts {
				damageByLoadout[weaponName] = 0
			}
		}
	}

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
		if combatLogger != nil {
			combatLogger.Info("No alive models to target")
		}
		return damageByLoadout, 0
	}

	if combatLogger != nil {
		combatLogger.Info("########################################")
		combatLogger.Info(fmt.Sprintf("Targeting model: %s (T%s, SV%s, ISV%s)",
			targetModel.Name,
			targetModel.Stats["T"],
			targetModel.Stats["SV"],
			targetModel.Stats["ISV"]))
	}

	// Iterate through all models in the attacker
	for _, model := range conflict.Attacker.Models {
		// Skip killed models
		if model.Killed >= model.Count {
			continue
		}

		// Calculate how many of this model are still alive
		aliveCount := model.Count - model.Killed

		// Iterate through each loadout/weapon for this model
		if model.Loadouts != nil {
			// First, check if this model has loadout options
			var weaponsToUse []string
			if len(conflict.Attacker.LoadoutOptions) > 0 {
				// If there are loadout options, use those
				for _, option := range conflict.Attacker.LoadoutOptions {
					if option.Type == "group" {
						weaponsToUse = option.Options
						break
					}
				}
			}

			// If no loadout options found, use all weapons
			if len(weaponsToUse) == 0 {
				for weaponName := range model.Loadouts {
					weaponsToUse = append(weaponsToUse, weaponName)
				}
			}

			// Now process each weapon in the loadout
			for _, weaponName := range weaponsToUse {
				weapon, exists := model.Loadouts[weaponName]
				if !exists {
					continue
				}

				// Add visual separator for new weapon attack
				if combatLogger != nil {
					combatLogger.Info(fmt.Sprintf("Starting attack with %s (%s)", weaponName, model.Name))
					combatLogger.Info("vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv")
				}

				if combatLogger != nil {
					combatLogger.Info(fmt.Sprintf("Starting weapon attack: %s (%s) targeting %s",
						weaponName,
						weapon.Type,
						targetModel.Name))
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
							if combatLogger != nil {
								combatLogger.Warn(fmt.Sprintf("Could not parse attacks '%s': %v", attacksStr, err))
							}
							continue
						}
					}
				} else {
					if combatLogger != nil {
						combatLogger.Warn("No attacks found for weapon, skipping")
					}
					continue
				}

				// Calculate total attacks (attacks per weapon * number of alive models)
				totalAttacks := attacks * aliveCount

				// Log attack count
				if combatLogger != nil {
					combatLogger.Info(fmt.Sprintf("Attack Count: %s - %s attacks, %d alive models, %d total attacks",
						weaponName,
						attacksStr,
						aliveCount,
						totalAttacks))
				}

				// PHASE 1: Roll for hits
				hits := 0
				sustainedHits := 0
				lethalHits := 0

				// Check if weapon has Torrent (auto-hit)
				keywords := weapon.GetStringCharacteristic("Keywords")
				isTorrent := strings.Contains(strings.ToLower(keywords), "torrent")

				if isTorrent {
					// Torrent weapons auto-hit
					if combatLogger != nil {
						combatLogger.Info(fmt.Sprintf("Hit Phase - Torrent: %d attacks auto-hit", totalAttacks))
					}
					hits = totalAttacks
				} else {
					// Get skill value (BS for ranged, WS for melee)
					var skillStr string
					if strings.Contains(strings.ToLower(weapon.Type), "ranged") {
						skillStr = weapon.GetStringCharacteristic("BS")
					} else if strings.Contains(strings.ToLower(weapon.Type), "melee") {
						skillStr = weapon.GetStringCharacteristic("WS")
					} else {
						if combatLogger != nil {
							combatLogger.Warn(fmt.Sprintf("Unknown weapon type '%s', assuming ranged", weapon.Type))
						}
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
							if combatLogger != nil {
								combatLogger.Warn(fmt.Sprintf("Could not parse skill '%s': %v", skillStr, err))
							}
							continue
						}
					} else {
						if combatLogger != nil {
							combatLogger.Warn("No skill value found for weapon, skipping")
						}
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

					if combatLogger != nil {
						combatLogger.Info(fmt.Sprintf("Hit Phase - Rolling: %d attacks, need %d+ to hit (base %d+ with %+d modifier)",
							totalAttacks,
							finalSkill,
							skillValue,
							weapon.Modifiers.HitMod))
					}

					// Roll for hits individually
					for i := 0; i < totalAttacks; i++ {
						roll := rollDice(1, 6)
						hit := roll >= finalSkill
						criticalHit := roll >= weapon.Modifiers.CritHit
						rerolled := false

						// Log initial hit roll
						if combatLogger != nil {
							combatLogger.Info(fmt.Sprintf("Initial Hit Roll: Attack %d rolled %d (need %d+), hit: %v, critical: %v",
								i+1,
								roll,
								finalSkill,
								hit,
								criticalHit))
						}

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
									combatLogger.Info(fmt.Sprintf("Miss Reroll: Attack %d rerolled %d (original %d), hit: %v, critical: %v",
										i+1,
										rerollResult,
										roll,
										hit,
										criticalHit))
								}

								roll = rerollResult // Update roll for logging
							}
						} else if weapon.Modifiers.CritHitFish && !criticalHit {
							// Critical hit fishing: reroll successful hits that weren't critical
							rerollResult := rollDice(1, 6)
							hit = rerollResult >= finalSkill
							criticalHit = rerollResult >= weapon.Modifiers.CritHit
							rerolled = true

							if combatLogger != nil {
								combatLogger.Info(fmt.Sprintf("Critical Hit Fishing Reroll: Attack %d rerolled %d (original %d), hit: %v, critical: %v",
									i+1,
									rerollResult,
									roll,
									hit,
									criticalHit))
							}

							roll = rerollResult // Update roll for logging
						}

						if hit {
							hits++

							// Handle critical hit effects
							if criticalHit {
								// Check for Sustained Hits
								keywordsLower := strings.ToLower(keywords)
								if strings.Contains(keywordsLower, "sustained hits") {
									// Parse sustained hits value
									// First split by commas, then by spaces
									keywordGroups := strings.Split(keywordsLower, ",")
									for _, group := range keywordGroups {
										group = strings.TrimSpace(group)
										if strings.Contains(group, "sustained hits") {
											// Split the group into words
											keywordParts := strings.Fields(group)
											for j, part := range keywordParts {
												if part == "sustained" && j+2 < len(keywordParts) && keywordParts[j+1] == "hits" {
													// Handle both formats: "Sustained Hits 1" and "Sustained Hits D3"
													sustainedValue := 1 // Default to 1 if not specified
													if j+2 < len(keywordParts) {
														sustainedStr := strings.TrimSpace(keywordParts[j+2])
														if strings.HasPrefix(sustainedStr, "d") {
															// Handle dice notation
															if sustainedDice, err := rollAndAdd(sustainedStr); err == nil {
																sustainedValue = sustainedDice
															}
														} else if val, err := strconv.Atoi(sustainedStr); err == nil {
															sustainedValue = val
														} else {
															if combatLogger != nil {
																combatLogger.Warn(fmt.Sprintf("Could not parse Sustained Hits value '%s'", sustainedStr))
															}
														}
													}

													if combatLogger != nil {
														combatLogger.Info(fmt.Sprintf("Sustained Hits Found: Weapon has 'Sustained Hits %d' keyword (from '%s')",
															sustainedValue,
															keywords))
													}

													sustainedHits += sustainedValue
													hits += sustainedValue

													if combatLogger != nil {
														combatLogger.Info(fmt.Sprintf("Sustained Hits Applied: Critical hit roll %d generated %d additional hits (total hits: %d)",
															roll,
															sustainedValue,
															hits))
													}
													break
												}
											}
										}
									}
								}

								// Check for Lethal Hits
								if strings.Contains(keywordsLower, "lethal hits") {
									lethalHits++
								}
							}
						}

						if combatLogger != nil {
							combatLogger.Info(fmt.Sprintf("Hit Roll Result: Attack %d final roll %d (need %d+), hit: %v, critical: %v, rerolled: %v, running hits: %d",
								i+1,
								roll,
								finalSkill,
								hit,
								criticalHit,
								rerolled,
								hits))
						}
					}

					if combatLogger != nil {
						combatLogger.Info(fmt.Sprintf("Hit Phase Complete: %d hits (including %d Sustained Hits, %d Lethal Hits)",
							hits,
							sustainedHits,
							lethalHits))
					}
				}

				// PHASE 2: Roll for wounds
				wounds, criticalWounds := conflict.rollWounds(hits, weapon, targetModel, lethalHits)

				// PHASE 3: Roll for saves and apply damage
				damageApplied := 0
				if wounds > 0 {
					damageApplied = conflict.rollSaves(wounds, criticalWounds, weapon, targetModel, targetModelIndex)
					totalDamage += damageApplied
					damageByLoadout[weaponName] = damageApplied
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

					combatLogger.Info(fmt.Sprintf("Weapon Attack Complete: Applied %d damage (cumulative: %d), %d models remaining with %d wounds",
						damageApplied,
						totalDamage,
						remainingModels,
						totalWoundsRemaining))
				}
			}
		}
	}

	if combatLogger != nil {
		combatLogger.Info(fmt.Sprintf("Combat Complete: Total damage %d",
			totalDamage))
	}
	return damageByLoadout, totalDamage
}

// Wound rolling method with detailed logging for each roll
func (conflict *UnitAttackSequence) rollWounds(hits int, weapon WeaponProfile, targetModel *ModelData, lethalHits int) (int, int) {
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
			if combatLogger != nil {
				combatLogger.Warn("Could not parse weapon strength",
					zap.String("strength_string", strengthStr),
					zap.Error(err))
			}
			return 0, 0
		}
	} else {
		if combatLogger != nil {
			combatLogger.Warn("No strength found for weapon")
		}
		return 0, 0
	}

	// Get target toughness
	toughnessStr := targetModel.Stats["T"]
	toughness := 0
	if toughnessStr != "" {
		if t, err := strconv.Atoi(toughnessStr); err == nil {
			toughness = t
		} else {
			if combatLogger != nil {
				combatLogger.Warn("Could not parse target toughness",
					zap.String("toughness_string", toughnessStr),
					zap.Error(err))
			}
			return 0, 0
		}
	} else {
		if combatLogger != nil {
			combatLogger.Warn("No toughness found for target")
		}
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

	// Check if weapon has Devastating Wounds keyword
	hasDevastatingWounds := strings.Contains(strings.ToLower(weapon.GetStringCharacteristic("Keywords")), "devastating wounds")

	if combatLogger != nil {
		combatLogger.Info("Wound Phase - Starting",
			zap.Int("hits", hits),
			zap.Int("lethal_hits", lethalHits),
			zap.Int("weapon_strength", strength),
			zap.Int("target_toughness", toughness),
			zap.Int("wound_threshold", finalWoundThreshold),
			zap.Int("base_threshold", woundThreshold),
			zap.Int("wound_modifier", weapon.Modifiers.WoundMod),
			zap.Bool("has_devastating_wounds", hasDevastatingWounds))
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
		criticalWound := hasDevastatingWounds && roll >= weapon.Modifiers.CritWound
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
				criticalWound = hasDevastatingWounds && rerollResult >= weapon.Modifiers.CritWound
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

	if combatLogger != nil {
		combatLogger.Info("Wound Phase Complete",
			zap.Int("total_wounds", wounds),
			zap.Int("devastating_wounds", criticalWounds),
			zap.Int("lethal_hits_autowound", lethalHits))
	}

	return wounds, criticalWounds
}

// Save rolling method with detailed logging for each roll
func (conflict *UnitAttackSequence) rollSaves(wounds int, criticalWounds int, weapon WeaponProfile, targetModel *ModelData, targetModelIndex int) int {
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
	devastatingDamage := 0

	// Process critical wounds first (they bypass saves)
	for i := 0; i < criticalWounds; i++ {
		if combatLogger != nil {
			combatLogger.Info("Devastating Wound",
				zap.Int("wound_number", i+1),
				zap.Bool("bypasses_save", true),
				zap.String("damage_characteristic", damageStr))
		}

		// Apply damage for Devastating Wound (no save allowed)
		damageAmount := conflict.applyDamage(targetModelIndex, damageStr, "devastating")
		devastatingDamage += damageAmount

		if combatLogger != nil {
			combatLogger.Info("Devastating Wound Damage Applied",
				zap.String("damage_amount", damageStr),
				zap.Int("actual_damage", damageAmount),
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

	failedSaveDamage := 0
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
			if combatLogger != nil {
				combatLogger.Info("Save Roll",
					zap.Int("wound_number", i+1+criticalWounds),
					zap.Int("roll", roll),
					zap.String("save_type", saveType),
					zap.Int("save_threshold", saveUsed),
					zap.Bool("saved", true))
			}
		} else {
			if combatLogger != nil {
				combatLogger.Info("Save Roll",
					zap.Int("wound_number", i+1+criticalWounds),
					zap.Int("roll", roll),
					zap.String("save_type", "failed"),
					zap.Int("save_threshold", 0),
					zap.Bool("saved", false))
			}

			// Apply damage for failed save
			damageAmount := conflict.applyDamage(targetModelIndex, damageStr)
			failedSaveDamage += damageAmount

			if combatLogger != nil {
				combatLogger.Info("Normal Wound Damage Applied",
					zap.String("damage_amount", damageStr),
					zap.Int("actual_damage", damageAmount),
					zap.Int("target_model_index", targetModelIndex),
					zap.String("target_model_name", targetModel.Name),
					zap.Bool("devastating_wound", false))
			}
		}
	}

	totalDamageApplied := devastatingDamage + failedSaveDamage

	if combatLogger != nil {
		combatLogger.Info("Save Phase Complete",
			zap.Int("total_damage_applied", totalDamageApplied),
			zap.Int("devastating_wound_damage", devastatingDamage),
			zap.Int("failed_save_damage", failedSaveDamage),
			zap.Int("wounds_saved", savedWounds))
	}

	return totalDamageApplied
}

// Apply abilities and weapon keywords that modify combat characteristics
func (conflict *UnitAttackSequence) applyAbilities() {
	if combatLogger != nil {
		combatLogger.Info("Starting ability processing",
			zap.String("attacker_unit", conflict.Attacker.Name),
			zap.Strings("attacker_abilities", conflict.Attacker.Abilities))
	}

	// Process attacker abilities
	for _, ability := range conflict.Attacker.Abilities {
		switch strings.ToLower(ability) {
		case "oath of moment":
			// Set reroll hits for all loadouts
			weaponsModified := 0
			for modelIndex := range conflict.Attacker.Models {
				for weaponName, weapon := range conflict.Attacker.Models[modelIndex].Loadouts {
					if !weapon.Modifiers.RerollHits { // Only modify if not already set
						weapon.Modifiers.RerollHits = true
						conflict.Attacker.Models[modelIndex].Loadouts[weaponName] = weapon
						weaponsModified++

						if combatLogger != nil {
							combatLogger.Info(fmt.Sprintf("Applied Oath of Moment to %s (%s): RerollHits = true (was false)",
								weaponName,
								conflict.Attacker.Models[modelIndex].Name))
						}
					}
				}
			}
			if combatLogger != nil {
				combatLogger.Info(fmt.Sprintf("Applied Oath of Moment: Modified %d weapons", weaponsModified))
			}

		case "stationary":
			// Heavy weapons get +1 to hit when stationary
			heavyWeaponsModified := 0
			for modelIndex := range conflict.Attacker.Models {
				for weaponName, weapon := range conflict.Attacker.Models[modelIndex].Loadouts {
					keywords := strings.ToLower(weapon.GetStringCharacteristic("Keywords"))
					if strings.Contains(keywords, "heavy") {
						previousHitMod := weapon.Modifiers.HitMod
						weapon.Modifiers.HitMod += 1
						conflict.Attacker.Models[modelIndex].Loadouts[weaponName] = weapon
						heavyWeaponsModified++

						if combatLogger != nil {
							combatLogger.Info(fmt.Sprintf("Applied Stationary to %s (%s): HitMod += 1 (was %d, now %d)",
								weaponName,
								conflict.Attacker.Models[modelIndex].Name,
								previousHitMod,
								weapon.Modifiers.HitMod))
						}
					}
				}
			}
			if combatLogger != nil {
				combatLogger.Info(fmt.Sprintf("Applied Stationary: Modified %d heavy weapons", heavyWeaponsModified))
			}

		case "rapid fire distance":
			// Rapid Fire weapons get additional shots equal to their rapid fire value
			rapidFireWeaponsModified := 0
			for modelIndex := range conflict.Attacker.Models {
				for weaponName, weapon := range conflict.Attacker.Models[modelIndex].Loadouts {
					keywords := strings.ToLower(weapon.GetStringCharacteristic("Keywords"))
					if strings.Contains(keywords, "rapid fire") {
						// Parse rapid fire value - look for "rapid fire X"
						keywordParts := strings.Fields(keywords)
						for j, part := range keywordParts {
							if part == "rapid" && j+2 < len(keywordParts) && keywordParts[j+1] == "fire" {
								if rapidFireValue, err := strconv.Atoi(keywordParts[j+2]); err == nil {
									// Get current attacks and add rapid fire bonus
									attacksStr := weapon.GetStringCharacteristic("A")
									if attacksStr != "" {
										if currentAttacks, err := strconv.Atoi(attacksStr); err == nil {
											newAttacks := currentAttacks + rapidFireValue
											weapon.Characteristics["A"] = strconv.Itoa(newAttacks)
											conflict.Attacker.Models[modelIndex].Loadouts[weaponName] = weapon
											rapidFireWeaponsModified++

											if combatLogger != nil {
												combatLogger.Info(fmt.Sprintf("Applied Rapid Fire Distance to %s (%s): Attacks += %d (was %d, now %d)",
													weaponName,
													conflict.Attacker.Models[modelIndex].Name,
													rapidFireValue,
													currentAttacks,
													newAttacks))
											}
										}
									}
								}
								break
							}
						}
					}
				}
			}
			if combatLogger != nil {
				combatLogger.Info(fmt.Sprintf("Applied Rapid Fire Distance: Modified %d rapid fire weapons", rapidFireWeaponsModified))
			}

		case "red rampage":
			// Red Rampage stratagem: Choose Lethal Hits OR Lance, or both with Battle-shocked
			// For simulation purposes, we'll implement as "gives both effects but unit becomes battle-shocked"
			// In a real game, this would be a player choice during combat
			// RED RAMPAGE ONLY AFFECTS MELEE WEAPONS (Fight Phase stratagem)
			weaponsModified := 0

			// Add Charged ability to enable Lance keyword
			exists, _ := stringExistsInSlice("Charged", conflict.Attacker.Abilities)
			if !exists {
				conflict.Attacker.Abilities = append(conflict.Attacker.Abilities, "Charged")
				if combatLogger != nil {
					combatLogger.Info(fmt.Sprintf("Added Charged ability to %s: Enables Lance keyword", conflict.Attacker.Name))
				}
			}

			for modelIndex := range conflict.Attacker.Models {
				for weaponName, weapon := range conflict.Attacker.Models[modelIndex].Loadouts {
					// Only apply to melee weapons
					weaponType := strings.ToLower(weapon.Type)
					if !strings.Contains(weaponType, "melee") {
						continue // Skip ranged weapons
					}

					// Apply both Lethal Hits and Lance effects (simulating "giving into the rampage")
					keywords := weapon.GetStringCharacteristic("Keywords")

					// Add Lethal Hits if not already present
					if !strings.Contains(strings.ToLower(keywords), "lethal hits") {
						if keywords != "" {
							keywords += ", Lethal Hits"
						} else {
							keywords = "Lethal Hits"
						}
					}

					// Add Lance if not already present
					if !strings.Contains(strings.ToLower(keywords), "lance") {
						if keywords != "" {
							keywords += ", Lance"
						} else {
							keywords = "Lance"
						}
					}

					weapon.Characteristics["Keywords"] = keywords

					// Apply Lance effect (+1 to wound) since unit has Charged
					// Only apply if not already modified
					if weapon.Modifiers.WoundMod == 0 {
						weapon.Modifiers.WoundMod = 1
					}

					conflict.Attacker.Models[modelIndex].Loadouts[weaponName] = weapon
					weaponsModified++

					if combatLogger != nil {
						combatLogger.Info(fmt.Sprintf("Applied Red Rampage to %s (%s): Added Lethal Hits + Lance (melee only), WoundMod = %d, Unit Battle-shocked",
							weaponName,
							conflict.Attacker.Models[modelIndex].Name,
							weapon.Modifiers.WoundMod))
					}
				}
			}
			if combatLogger != nil {
				combatLogger.Info(fmt.Sprintf("Applied Red Rampage: Modified %d melee weapons", weaponsModified))
			}

		case "crithitfish":
			critFishWeaponsModified := 0
			for modelIndex := range conflict.Attacker.Models {
				for weaponName, weapon := range conflict.Attacker.Models[modelIndex].Loadouts {
					if !weapon.Modifiers.CritHitFish {
						weapon.Modifiers.CritHitFish = true
						conflict.Attacker.Models[modelIndex].Loadouts[weaponName] = weapon
						critFishWeaponsModified++

						if combatLogger != nil {
							combatLogger.Info(fmt.Sprintf("Applied CritHitFish to %s (%s): CritHitFish = true",
								weaponName,
								conflict.Attacker.Models[modelIndex].Name))
						}
					}
				}
			}
			if combatLogger != nil {
				combatLogger.Info(fmt.Sprintf("Applied CritHitFish: Modified %d weapons", critFishWeaponsModified))
			}
		}
	}

	// Process defender abilities
	for _, ability := range conflict.Defender.Abilities {
		switch strings.ToLower(ability) {
		case "stealth":
			// Stealth subtracts 1 from hit rolls for ranged attacks
			rangedWeaponsModified := 0
			for modelIndex := range conflict.Attacker.Models {
				for weaponName, weapon := range conflict.Attacker.Models[modelIndex].Loadouts {
					weaponType := strings.ToLower(weapon.Type)
					if strings.Contains(weaponType, "ranged") {
						previousHitMod := weapon.Modifiers.HitMod
						weapon.Modifiers.HitMod -= 1
						conflict.Attacker.Models[modelIndex].Loadouts[weaponName] = weapon
						rangedWeaponsModified++

						if combatLogger != nil {
							combatLogger.Info(fmt.Sprintf("Applied Stealth to %s (%s): HitMod -= 1 (was %d, now %d)",
								weaponName,
								conflict.Attacker.Models[modelIndex].Name,
								previousHitMod,
								weapon.Modifiers.HitMod))
						}
					}
				}
			}
			if combatLogger != nil {
				combatLogger.Info(fmt.Sprintf("Applied Stealth: Modified %d ranged weapons", rangedWeaponsModified))
			}
		}
	}

	// Process weapon-specific abilities (Twin-linked)
	twinLinkedWeaponsModified := 0
	for modelIndex := range conflict.Attacker.Models {
		for weaponName, weapon := range conflict.Attacker.Models[modelIndex].Loadouts {
			weaponNameLower := strings.ToLower(weaponName)
			keywords := strings.ToLower(weapon.GetStringCharacteristic("Keywords"))

			// Check for Twin-linked in weapon name or keywords
			if strings.Contains(weaponNameLower, "twin-linked") || strings.Contains(keywords, "twin-linked") {
				if !weapon.Modifiers.RerollWounds { // Only modify if not already set
					weapon.Modifiers.RerollWounds = true
					conflict.Attacker.Models[modelIndex].Loadouts[weaponName] = weapon
					twinLinkedWeaponsModified++

					if combatLogger != nil {
						combatLogger.Info(fmt.Sprintf("Applied Twin-linked to %s (%s): RerollWounds = true (was false)",
							weaponName,
							conflict.Attacker.Models[modelIndex].Name))
					}
				}
			}
		}
	}

	if combatLogger != nil {
		combatLogger.Info(fmt.Sprintf("Ability processing complete for %s: Modified %d twin-linked weapons",
			conflict.Attacker.Name,
			twinLinkedWeaponsModified))
	}
}
