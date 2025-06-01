package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// Core XML structures following BattleScribe schema
type Catalogue struct {
	XMLName                xml.Name         `xml:"catalogue"`
	ID                     string           `xml:"id,attr"`
	Name                   string           `xml:"name,attr"`
	GameSystemId           string           `xml:"gameSystemId,attr"`
	GameSystemRevision     string           `xml:"gameSystemRevision,attr"`
	SharedSelectionEntries []SelectionEntry `xml:"sharedSelectionEntries>selectionEntry"`
	SelectionEntries       []SelectionEntry `xml:"selectionEntries>selectionEntry"`
	CategoryEntries        []CategoryEntry  `xml:"categoryEntries>categoryEntry"`
	SharedProfiles         []Profile        `xml:"sharedProfiles>profile"`
	SharedRules            []Rule           `xml:"sharedRules>rule"`
}

type SelectionEntry struct {
	ID                   string                `xml:"id,attr"`
	Name                 string                `xml:"name,attr"`
	Type                 string                `xml:"type,attr"`
	Hidden               string                `xml:"hidden,attr"`
	Profiles             []Profile             `xml:"profiles>profile"`
	Rules                []Rule                `xml:"rules>rule"`
	InfoLinks            []InfoLink            `xml:"infoLinks>infoLink"`
	SelectionEntries     []SelectionEntry      `xml:"selectionEntries>selectionEntry"`
	SelectionEntryGroups []SelectionEntryGroup `xml:"selectionEntryGroups>selectionEntryGroup"`
	CategoryLinks        []CategoryLink        `xml:"categoryLinks>categoryLink"`
	Constraints          []Constraint          `xml:"constraints>constraint"`
	Costs                []Cost                `xml:"costs>cost"`
	Modifiers            []Modifier            `xml:"modifiers>modifier"`
	EntryLinks           []EntryLink           `xml:"entryLinks>entryLink"`
}

type SelectionEntryGroup struct {
	ID                      string                `xml:"id,attr"`
	Name                    string                `xml:"name,attr"`
	Hidden                  string                `xml:"hidden,attr"`
	DefaultSelectionEntryId string                `xml:"defaultSelectionEntryId,attr"`
	SelectionEntries        []SelectionEntry      `xml:"selectionEntries>selectionEntry"`
	SelectionEntryGroups    []SelectionEntryGroup `xml:"selectionEntryGroups>selectionEntryGroup"`
	EntryLinks              []EntryLink           `xml:"entryLinks>entryLink"`
	Constraints             []Constraint          `xml:"constraints>constraint"`
	Profiles                []Profile             `xml:"profiles>profile"`
	Rules                   []Rule                `xml:"rules>rule"`
	InfoLinks               []InfoLink            `xml:"infoLinks>infoLink"`
	CategoryLinks           []CategoryLink        `xml:"categoryLinks>categoryLink"`
}

type Profile struct {
	ID              string           `xml:"id,attr"`
	Name            string           `xml:"name,attr"`
	TypeId          string           `xml:"typeId,attr"`
	TypeName        string           `xml:"typeName,attr"`
	Hidden          string           `xml:"hidden,attr"`
	Characteristics []Characteristic `xml:"characteristics>characteristic"`
}

type Characteristic struct {
	ID     string `xml:"id,attr"`
	Name   string `xml:"name,attr"`
	TypeId string `xml:"typeId,attr"`
	Value  string `xml:",chardata"`
}

type Rule struct {
	ID          string `xml:"id,attr"`
	Name        string `xml:"name,attr"`
	Hidden      string `xml:"hidden,attr"`
	Description string `xml:"description"`
}

type InfoLink struct {
	ID       string `xml:"id,attr"`
	Name     string `xml:"name,attr"`
	Hidden   string `xml:"hidden,attr"`
	Type     string `xml:"type,attr"`
	TargetId string `xml:"targetId,attr"`
}

type CategoryLink struct {
	ID       string `xml:"id,attr"`
	Name     string `xml:"name,attr"`
	Hidden   string `xml:"hidden,attr"`
	TargetId string `xml:"targetId,attr"`
	Primary  string `xml:"primary,attr"`
}

type CategoryEntry struct {
	ID     string `xml:"id,attr"`
	Name   string `xml:"name,attr"`
	Hidden string `xml:"hidden,attr"`
}

type Constraint struct {
	ID    string `xml:"id,attr"`
	Type  string `xml:"type,attr"`
	Value string `xml:"value,attr"`
	Field string `xml:"field,attr"`
	Scope string `xml:"scope,attr"`
}

type Cost struct {
	ID     string `xml:"id,attr"`
	Name   string `xml:"name,attr"`
	TypeId string `xml:"typeId,attr"`
	Value  string `xml:"value,attr"`
}

type Modifier struct {
	ID    string `xml:"id,attr"`
	Type  string `xml:"type,attr"`
	Field string `xml:"field,attr"`
	Value string `xml:"value,attr"`
}

type EntryLink struct {
	ID       string `xml:"id,attr"`
	Name     string `xml:"name,attr"`
	Hidden   string `xml:"hidden,attr"`
	Type     string `xml:"type,attr"`
	TargetId string `xml:"targetId,attr"`
}

// Our output structures
type UnitData struct {
	Name           string          `yaml:"name"`
	Type           string          `yaml:"type"`
	Cost           int             `yaml:"cost"`
	Abilities      []string        `yaml:"abilities,omitempty"`
	Keywords       []string        `yaml:"keywords,omitempty"`
	Models         []ModelData     `yaml:"models"`
	LoadoutOptions []LoadoutOption `yaml:"loadout_options,omitempty"`
}

type ModelData struct {
	Name        string                   `yaml:"name"`
	Count       int                      `yaml:"count"`
	Stats       map[string]string        `yaml:"stats,omitempty"`
	BaseLoadout []string                 `yaml:"base_loadout,omitempty"`
	Loadouts    map[string]WeaponProfile `yaml:"loadouts,omitempty"`
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
}

func main() {
	unitName := flag.String("unit", "Brutalis Dreadnought", "Name of the unit to extract")
	combineMode := flag.Bool("combine", false, "Combine two units: input1.yaml input2.yaml output.yaml")
	showHelp := flag.Bool("help", false, "Show usage information")
	flag.Parse()

	if *showHelp {
		fmt.Println("Library Builder - Warhammer 40k Unit Extractor")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  Extract a unit from BattleScribe data:")
		fmt.Println("    go run main.go --unit \"Unit Name\"")
		fmt.Println()
		fmt.Println("  Combine two existing library units:")
		fmt.Println("    go run main.go --combine input1.yaml input2.yaml output.yaml")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run main.go --unit \"Captain with Jump Pack\"")
		fmt.Println("  go run main.go --combine captain_with_jump_pack.yaml bladeguard_veteran_squad.yaml combined_force.yaml")
		fmt.Println()
		return
	}

	// Check if we're combining units
	if *combineMode {
		args := flag.Args()
		if len(args) != 3 {
			log.Fatalf("Combine mode requires exactly 3 arguments: input1.yaml input2.yaml output.yaml")
		}
		err := combineUnitFiles(args[0], args[1], args[2])
		if err != nil {
			log.Fatalf("Error combining units: %v", err)
		}
		return
	}

	fmt.Printf("Extracting unit: %s\n", *unitName)

	// Find catalog files
	catalogFiles, err := findCatalogFiles("battlescribe-data-10e")
	if err != nil {
		log.Fatalf("Error finding catalog files: %v", err)
	}

	fmt.Printf("Found %d catalog files\n", len(catalogFiles))

	var foundUnit *UnitData
	var sourceCatalog string

	// Search through catalogs for the unit
	for _, catalogFile := range catalogFiles {
		unit, err := extractUnitFromCatalog(catalogFile, *unitName)
		if err != nil {
			log.Printf("Error processing %s: %v", catalogFile, err)
			continue
		}
		if unit != nil {
			foundUnit = unit
			sourceCatalog = catalogFile
			break
		}
	}

	if foundUnit == nil {
		log.Fatalf("Unit '%s' not found in any catalog", *unitName)
	}

	fmt.Printf("Found unit in: %s\n", sourceCatalog)

	// Write to YAML file
	outputFile := fmt.Sprintf("library/%s.yaml", strings.ReplaceAll(strings.ToLower(*unitName), " ", "_"))
	err = writeUnitYAML(foundUnit, outputFile)
	if err != nil {
		log.Fatalf("Error writing YAML: %v", err)
	}

	fmt.Printf("Unit data written to: %s\n", outputFile)
	fmt.Printf("\nUnit Summary:\n")
	fmt.Printf("  Name: %s\n", foundUnit.Name)
	fmt.Printf("  Type: %s\n", foundUnit.Type)
	fmt.Printf("  Cost: %d\n", foundUnit.Cost)
	fmt.Printf("  Models: %d\n", len(foundUnit.Models))
	fmt.Printf("  Loadout Options: %d\n", len(foundUnit.LoadoutOptions))
	fmt.Printf("  Abilities: %d\n", len(foundUnit.Abilities))
	fmt.Printf("  Keywords: %d\n", len(foundUnit.Keywords))
}

func findCatalogFiles(dataDir string) ([]string, error) {
	var catalogFiles []string

	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), ".cat") {
			catalogFiles = append(catalogFiles, path)
		}
		return nil
	})

	return catalogFiles, err
}

func extractUnitFromCatalog(catalogFile, unitName string) (*UnitData, error) {
	data, err := ioutil.ReadFile(catalogFile)
	if err != nil {
		return nil, err
	}

	var catalog Catalogue
	err = xml.Unmarshal(data, &catalog)
	if err != nil {
		return nil, err
	}

	// Search in shared selection entries first
	if unit := findSelectionEntry(catalog.SharedSelectionEntries, unitName); unit != nil {
		return processSelectionEntry(unit, &catalog), nil
	}

	// Search in regular selection entries
	if unit := findSelectionEntry(catalog.SelectionEntries, unitName); unit != nil {
		return processSelectionEntry(unit, &catalog), nil
	}

	return nil, nil
}

func findSelectionEntry(entries []SelectionEntry, name string) *SelectionEntry {
	for i := range entries {
		if strings.EqualFold(entries[i].Name, name) {
			return &entries[i]
		}
	}
	return nil
}

func processSelectionEntry(entry *SelectionEntry, catalog *Catalogue) *UnitData {
	unit := &UnitData{
		Name:   entry.Name,
		Type:   getUnitType(entry.Type),
		Models: []ModelData{},
	}

	// Extract cost
	unit.Cost = extractCost(entry.Costs)

	// Extract keywords from category links
	unit.Keywords = extractKeywords(entry.CategoryLinks, catalog.CategoryEntries)

	// Extract abilities from rules and profiles
	abilitiesFromRules := extractAbilities(entry.Rules, entry.InfoLinks, catalog)
	abilitiesFromProfiles := extractAbilitiesFromProfiles(entry.Profiles)

	// Combine and deduplicate abilities
	abilityMap := make(map[string]bool)
	for _, ability := range abilitiesFromRules {
		abilityMap[ability] = true
	}
	for _, ability := range abilitiesFromProfiles {
		abilityMap[ability] = true
	}

	unit.Abilities = []string{}
	for ability := range abilityMap {
		unit.Abilities = append(unit.Abilities, ability)
	}
	sort.Strings(unit.Abilities)

	// Extract unit stats from profiles
	extractUnitStats(unit, entry.Profiles)

	// Process models and loadouts
	processModelsAndLoadouts(unit, entry, catalog)

	// After all processing is done, add invulnerable save to model stats if the unit has one
	for _, ability := range unit.Abilities {
		if strings.Contains(strings.ToLower(ability), "invulnerable save") {
			// Extract the save value from the ability (e.g., "Invulnerable Save (4+)" -> "4+")
			if strings.Contains(ability, "(") && strings.Contains(ability, ")") {
				start := strings.Index(ability, "(") + 1
				end := strings.Index(ability, ")")
				if start < end {
					invulnSave := ability[start:end]
					// Add ISV to all models in the unit
					for i := range unit.Models {
						if unit.Models[i].Stats == nil {
							unit.Models[i].Stats = make(map[string]string)
						}
						unit.Models[i].Stats["ISV"] = invulnSave
					}
				}
			}
			break // Only process the first invulnerable save ability found
		}
	}

	return unit
}

func getUnitType(entryType string) string {
	switch strings.ToLower(entryType) {
	case "unit":
		return "unit"
	case "model":
		return "model"
	case "upgrade":
		return "upgrade"
	default:
		return "unit"
	}
}

func extractCost(costs []Cost) int {
	for _, cost := range costs {
		if cost.Name == "pts" || cost.Name == "points" || strings.Contains(strings.ToLower(cost.Name), "point") {
			if value, err := strconv.Atoi(cost.Value); err == nil {
				return value
			}
		}
	}
	return 0
}

func extractKeywords(categoryLinks []CategoryLink, categories []CategoryEntry) []string {
	var keywords []string
	categoryMap := make(map[string]string)

	// Build category map
	for _, cat := range categories {
		categoryMap[cat.ID] = cat.Name
	}

	// Extract keyword names
	for _, link := range categoryLinks {
		if name, exists := categoryMap[link.TargetId]; exists {
			keywords = append(keywords, name)
		} else if link.Name != "" {
			keywords = append(keywords, link.Name)
		}
	}

	sort.Strings(keywords)
	return keywords
}

func extractAbilities(rules []Rule, infoLinks []InfoLink, catalog *Catalogue) []string {
	var abilities []string

	// Extract from local rules
	for _, rule := range rules {
		if rule.Hidden != "true" && rule.Name != "" {
			ability := rule.Name
			// Check if the description contains a save value (like "4+")
			if strings.Contains(strings.ToLower(rule.Name), "invulnerable") && rule.Description != "" {
				// Look for save values in the description with flexible pattern matching
				desc := rule.Description
				if strings.Contains(desc, "4+") || desc == "4+" {
					ability = rule.Name + " (4+)"
				} else if strings.Contains(desc, "5+") || desc == "5+" {
					ability = rule.Name + " (5+)"
				} else if strings.Contains(desc, "3+") || desc == "3+" {
					ability = rule.Name + " (3+)"
				} else if strings.Contains(desc, "6+") || desc == "6+" {
					ability = rule.Name + " (6+)"
				} else if strings.Contains(desc, "2+") || desc == "2+" {
					ability = rule.Name + " (2+)"
				}
			}
			abilities = append(abilities, ability)
		}
	}

	// Extract from info links (references to shared rules)
	for _, link := range infoLinks {
		if link.Hidden != "true" && link.Name != "" {
			ability := link.Name
			// Try to find the referenced rule in shared rules to get its description
			for _, sharedRule := range catalog.SharedRules {
				if sharedRule.ID == link.TargetId && sharedRule.Description != "" {
					// Check if this is an invulnerable save with a value
					if strings.Contains(strings.ToLower(link.Name), "invulnerable") {
						desc := sharedRule.Description
						if strings.Contains(desc, "4+") || desc == "4+" {
							ability = link.Name + " (4+)"
						} else if strings.Contains(desc, "5+") || desc == "5+" {
							ability = link.Name + " (5+)"
						} else if strings.Contains(desc, "3+") || desc == "3+" {
							ability = link.Name + " (3+)"
						} else if strings.Contains(desc, "6+") || desc == "6+" {
							ability = link.Name + " (6+)"
						} else if strings.Contains(desc, "2+") || desc == "2+" {
							ability = link.Name + " (2+)"
						}
					}
					break
				}
			}

			// Also check shared profiles for invulnerable saves
			if strings.Contains(strings.ToLower(link.Name), "invulnerable") {
				for _, sharedProfile := range catalog.SharedProfiles {
					if sharedProfile.ID == link.TargetId {
						for _, char := range sharedProfile.Characteristics {
							if strings.Contains(strings.ToLower(char.Name), "description") {
								desc := char.Value
								if strings.Contains(desc, "4+") || desc == "4+" {
									ability = link.Name + " (4+)"
								} else if strings.Contains(desc, "5+") || desc == "5+" {
									ability = link.Name + " (5+)"
								} else if strings.Contains(desc, "3+") || desc == "3+" {
									ability = link.Name + " (3+)"
								} else if strings.Contains(desc, "6+") || desc == "6+" {
									ability = link.Name + " (6+)"
								} else if strings.Contains(desc, "2+") || desc == "2+" {
									ability = link.Name + " (2+)"
								}
								break
							}
						}
						break
					}
				}
			}

			abilities = append(abilities, ability)
		}
	}

	sort.Strings(abilities)
	return abilities
}

// New function to extract abilities from profiles as well
func extractAbilitiesFromProfiles(profiles []Profile) []string {
	var abilities []string

	for _, profile := range profiles {
		if profile.TypeName == "Abilities" && profile.Hidden != "true" {
			ability := profile.Name
			// Check if this is an invulnerable save ability
			if strings.Contains(strings.ToLower(profile.Name), "invulnerable") {
				for _, char := range profile.Characteristics {
					if strings.Contains(strings.ToLower(char.Name), "description") {
						desc := char.Value
						if strings.Contains(desc, "4+") || desc == "4+" {
							ability = profile.Name + " (4+)"
						} else if strings.Contains(desc, "5+") || desc == "5+" {
							ability = profile.Name + " (5+)"
						} else if strings.Contains(desc, "3+") || desc == "3+" {
							ability = profile.Name + " (3+)"
						} else if strings.Contains(desc, "6+") || desc == "6+" {
							ability = profile.Name + " (6+)"
						} else if strings.Contains(desc, "2+") || desc == "2+" {
							ability = profile.Name + " (2+)"
						}
						break
					}
				}
			}
			abilities = append(abilities, ability)
		}
	}

	return abilities
}

func extractUnitStats(unit *UnitData, profiles []Profile) {
	for _, profile := range profiles {
		if profile.TypeName == "Unit" && profile.Hidden != "true" {
			stats := make(map[string]string)
			for _, char := range profile.Characteristics {
				if char.Value != "" {
					stats[char.Name] = char.Value
				}
			}
			if len(stats) > 0 {
				model := ModelData{
					Name:     profile.Name,
					Count:    1,
					Stats:    stats,
					Loadouts: make(map[string]WeaponProfile),
				}
				unit.Models = append(unit.Models, model)
			}
		}
		// Note: Weapon profiles are now handled per-model, not globally
	}
}

func processModelsAndLoadouts(unit *UnitData, entry *SelectionEntry, catalog *Catalogue) {
	// Build a map of shared entries for quick lookup
	sharedMap := buildSharedEntryMap(catalog)

	// Process selection entry groups for loadout options
	for _, group := range entry.SelectionEntryGroups {
		if group.Hidden != "true" {
			// Check if this group contains model entries (like Bladeguard Veterans)
			hasModels := false
			for _, subEntry := range group.SelectionEntries {
				if subEntry.Type == "model" {
					hasModels = true
					break
				}
			}

			if hasModels {
				// This group contains models - extract them as actual models, not loadout options
				for _, subEntry := range group.SelectionEntries {
					if subEntry.Hidden != "true" && subEntry.Type == "model" {
						var model ModelData
						model.Loadouts = make(map[string]WeaponProfile)

						// Extract model stats from Unit profiles
						for _, profile := range subEntry.Profiles {
							if profile.TypeName == "Unit" && profile.Hidden != "true" {
								stats := make(map[string]string)
								for _, char := range profile.Characteristics {
									if char.Value != "" {
										stats[char.Name] = char.Value
									}
								}
								if len(stats) > 0 {
									model.Name = profile.Name
									model.Count = 1
									model.Stats = stats
								}
							} else if profile.TypeName == "Weapon" || profile.TypeName == "Ranged Weapons" || profile.TypeName == "Melee Weapons" {
								if profile.Hidden != "true" {
									weaponProfile := WeaponProfile{
										Name:            profile.Name,
										Type:            profile.TypeName,
										Characteristics: make(map[string]string),
									}
									for _, char := range profile.Characteristics {
										if char.Value != "" {
											weaponProfile.Characteristics[char.Name] = char.Value
										}
									}
									model.Loadouts[profile.Name] = weaponProfile
								}
							}
						}

						// Process weapon options within this model (selection entry groups)
						for _, modelGroup := range subEntry.SelectionEntryGroups {
							if modelGroup.Hidden != "true" {
								// Process sub-entries in the weapon group
								for _, option := range modelGroup.SelectionEntries {
									if option.Hidden != "true" {
										// Extract weapons from the option
										for _, profile := range option.Profiles {
											if profile.TypeName == "Weapon" || profile.TypeName == "Ranged Weapons" || profile.TypeName == "Melee Weapons" {
												if profile.Hidden != "true" {
													weaponProfile := WeaponProfile{
														Name:            profile.Name,
														Type:            profile.TypeName,
														Characteristics: make(map[string]string),
													}
													for _, char := range profile.Characteristics {
														if char.Value != "" {
															weaponProfile.Characteristics[char.Name] = char.Value
														}
													}
													model.Loadouts[profile.Name] = weaponProfile
												}
											}
										}
									}
								}

								// Process entry links in the weapon group
								for _, link := range modelGroup.EntryLinks {
									if link.Hidden != "true" {
										if sharedEntry, exists := sharedMap[link.TargetId]; exists {
											for _, profile := range sharedEntry.Profiles {
												if profile.TypeName == "Weapon" || profile.TypeName == "Ranged Weapons" || profile.TypeName == "Melee Weapons" {
													if profile.Hidden != "true" {
														weaponProfile := WeaponProfile{
															Name:            profile.Name,
															Type:            profile.TypeName,
															Characteristics: make(map[string]string),
														}
														for _, char := range profile.Characteristics {
															if char.Value != "" {
																weaponProfile.Characteristics[char.Name] = char.Value
															}
														}
														model.Loadouts[profile.Name] = weaponProfile
													}
												}
											}
										}
									}
								}
							}
						}

						// Also process any entry links in the model for weapons
						for _, entryLink := range subEntry.EntryLinks {
							if entryLink.Hidden != "true" {
								if sharedEntry, exists := sharedMap[entryLink.TargetId]; exists {
									for _, profile := range sharedEntry.Profiles {
										if profile.TypeName == "Weapon" || profile.TypeName == "Ranged Weapons" || profile.TypeName == "Melee Weapons" {
											if profile.Hidden != "true" {
												weaponProfile := WeaponProfile{
													Name:            profile.Name,
													Type:            profile.TypeName,
													Characteristics: make(map[string]string),
												}
												for _, char := range profile.Characteristics {
													if char.Value != "" {
														weaponProfile.Characteristics[char.Name] = char.Value
													}
												}
												model.Loadouts[profile.Name] = weaponProfile
											}
										}
									}
								}
							}
						}

						if model.Name != "" {
							unit.Models = append(unit.Models, model)
						}
					}
				}
			} else {
				// This is a regular loadout group
				loadoutOption := LoadoutOption{
					Name: group.Name,
					Type: "group",
				}

				// Extract options from the group
				var options []string
				for _, subEntry := range group.SelectionEntries {
					if subEntry.Hidden != "true" {
						options = append(options, subEntry.Name)

						// Note: Weapon profiles are now handled per-model, not globally
					}
				}

				// Process entry links in the group
				for _, entryLink := range group.EntryLinks {
					if entryLink.Hidden != "true" {
						if sharedEntry, exists := sharedMap[entryLink.TargetId]; exists {
							options = append(options, sharedEntry.Name)

							// Note: Weapon profiles from shared entries are now handled per-model
						}
					}
				}

				// Process nested selection entry groups
				for _, subGroup := range group.SelectionEntryGroups {
					if subGroup.Hidden != "true" {
						subLoadoutOption := LoadoutOption{
							Name: subGroup.Name,
							Type: "group",
						}

						var subOptions []string
						for _, subEntry := range subGroup.SelectionEntries {
							if subEntry.Hidden != "true" {
								subOptions = append(subOptions, subEntry.Name)

								// Note: Weapon profiles are now handled per-model, not globally
							}
						}

						// Process entry links in the sub-group
						for _, entryLink := range subGroup.EntryLinks {
							if entryLink.Hidden != "true" {
								if sharedEntry, exists := sharedMap[entryLink.TargetId]; exists {
									subOptions = append(subOptions, sharedEntry.Name)

									// Note: Weapon profiles from shared entries are now handled per-model
								}
							}
						}

						subLoadoutOption.Options = subOptions
						if len(subOptions) > 0 {
							unit.LoadoutOptions = append(unit.LoadoutOptions, subLoadoutOption)
						}
					}
				}

				loadoutOption.Options = options
				if len(options) > 0 {
					unit.LoadoutOptions = append(unit.LoadoutOptions, loadoutOption)
				}
			}
		}
	}

	// Process direct selection entries for models/upgrades
	for _, subEntry := range entry.SelectionEntries {
		if subEntry.Hidden != "true" {
			if subEntry.Type == "model" {
				// This is a model variant
				stats := make(map[string]string)
				for _, profile := range subEntry.Profiles {
					if profile.TypeName == "Unit" {
						for _, char := range profile.Characteristics {
							if char.Value != "" {
								stats[char.Name] = char.Value
							}
						}
					}
				}
				if len(stats) > 0 {
					model := ModelData{
						Name:  subEntry.Name,
						Count: 1,
						Stats: stats,
					}
					unit.Models = append(unit.Models, model)
				}
			} else if subEntry.Type == "upgrade" {
				// This is an upgrade option
				loadoutOption := LoadoutOption{
					Name: subEntry.Name,
					Type: "upgrade",
				}
				unit.LoadoutOptions = append(unit.LoadoutOptions, loadoutOption)
			}
		}
	}

	// Process entry links at the main entry level
	for _, entryLink := range entry.EntryLinks {
		if entryLink.Hidden != "true" {
			if sharedEntry, exists := sharedMap[entryLink.TargetId]; exists {
				// Check if this is a weapon entry
				for _, profile := range sharedEntry.Profiles {
					if profile.TypeName == "Weapon" || profile.TypeName == "Ranged Weapons" || profile.TypeName == "Melee Weapons" {
						if profile.Hidden != "true" {
							// Note: Weapons are now handled per-model rather than globally

							// Add to base loadout if it's a standard weapon
							if len(unit.Models) > 0 {
								weaponName := profile.Name
								if unit.Models[0].BaseLoadout == nil {
									unit.Models[0].BaseLoadout = []string{}
								}
								unit.Models[0].BaseLoadout = append(unit.Models[0].BaseLoadout, weaponName)

								// Also add the weapon profile to all models
								weaponProfile := WeaponProfile{
									Name:            profile.Name,
									Type:            profile.TypeName,
									Characteristics: make(map[string]string),
								}
								for _, char := range profile.Characteristics {
									if char.Value != "" {
										weaponProfile.Characteristics[char.Name] = char.Value
									}
								}

								// Add to all models
								for i := range unit.Models {
									if unit.Models[i].Loadouts == nil {
										unit.Models[i].Loadouts = make(map[string]WeaponProfile)
									}
									unit.Models[i].Loadouts[profile.Name] = weaponProfile
								}
							}
						}
					}
				}
			}
		}
	}

	// If no models were found, create a default model from the main entry
	if len(unit.Models) == 0 {
		for _, profile := range entry.Profiles {
			if profile.TypeName == "Unit" && profile.Hidden != "true" {
				stats := make(map[string]string)
				for _, char := range profile.Characteristics {
					if char.Value != "" {
						stats[char.Name] = char.Value
					}
				}
				if len(stats) > 0 {
					model := ModelData{
						Name:  entry.Name,
						Count: 1,
						Stats: stats,
					}
					unit.Models = append(unit.Models, model)
					break
				}
			}
		}
	}

	// Process loadout options and add all weapons to models
	for _, loadoutOption := range unit.LoadoutOptions {
		for _, optionName := range loadoutOption.Options {
			// Try to find this option in our shared map (which includes recursive entries)
			for _, sharedEntry := range sharedMap {
				// Match by name or if the option name contains the entry name or vice versa
				if sharedEntry.Name == optionName ||
					strings.Contains(sharedEntry.Name, optionName) ||
					strings.Contains(optionName, sharedEntry.Name) {

					for _, profile := range sharedEntry.Profiles {
						if profile.TypeName == "Weapon" || profile.TypeName == "Ranged Weapons" || profile.TypeName == "Melee Weapons" {
							if profile.Hidden != "true" {
								weaponProfile := WeaponProfile{
									Name:            profile.Name,
									Type:            profile.TypeName,
									Characteristics: make(map[string]string),
								}
								for _, char := range profile.Characteristics {
									if char.Value != "" {
										weaponProfile.Characteristics[char.Name] = char.Value
									}
								}
								// Add to the first model (assuming single model units like Brutalis)
								if len(unit.Models) > 0 {
									if unit.Models[0].Loadouts == nil {
										unit.Models[0].Loadouts = make(map[string]WeaponProfile)
									}
									unit.Models[0].Loadouts[profile.Name] = weaponProfile
								}
							}
						}
					}

					// Also process nested selection entries within the loadout option
					for _, nestedEntry := range sharedEntry.SelectionEntries {
						if nestedEntry.Hidden != "true" {
							for _, profile := range nestedEntry.Profiles {
								if profile.TypeName == "Weapon" || profile.TypeName == "Ranged Weapons" || profile.TypeName == "Melee Weapons" {
									if profile.Hidden != "true" {
										weaponProfile := WeaponProfile{
											Name:            profile.Name,
											Type:            profile.TypeName,
											Characteristics: make(map[string]string),
										}
										for _, char := range profile.Characteristics {
											if char.Value != "" {
												weaponProfile.Characteristics[char.Name] = char.Value
											}
										}
										if len(unit.Models) > 0 {
											if unit.Models[0].Loadouts == nil {
												unit.Models[0].Loadouts = make(map[string]WeaponProfile)
											}
											unit.Models[0].Loadouts[profile.Name] = weaponProfile
										}
									}
								}
							}
						}
					}

					// Also process entry links within the loadout option
					for _, entryLink := range sharedEntry.EntryLinks {
						if entryLink.Hidden != "true" {
							if linkedEntry, exists := sharedMap[entryLink.TargetId]; exists {
								for _, profile := range linkedEntry.Profiles {
									if profile.TypeName == "Weapon" || profile.TypeName == "Ranged Weapons" || profile.TypeName == "Melee Weapons" {
										if profile.Hidden != "true" {
											weaponProfile := WeaponProfile{
												Name:            profile.Name,
												Type:            profile.TypeName,
												Characteristics: make(map[string]string),
											}
											for _, char := range profile.Characteristics {
												if char.Value != "" {
													weaponProfile.Characteristics[char.Name] = char.Value
												}
											}
											if len(unit.Models) > 0 {
												if unit.Models[0].Loadouts == nil {
													unit.Models[0].Loadouts = make(map[string]WeaponProfile)
												}
												unit.Models[0].Loadouts[profile.Name] = weaponProfile
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Special handling for composite loadout options like "Melee and Pistol"
	if strings.Contains(unit.Name, "Captain") {
		// Search for common Captain weapons that might be in the "Melee and Pistol" loadout
		captainWeapons := []string{
			"Bolt Pistol", "Heavy Bolt Pistol", "Neo-volkite Pistol", "Plasma Pistol",
			"Master-crafted Bolt Rifle", "Master-crafted Power Weapon", "Power Fist",
			"Chainsword", "Power Sword", "Lightning Claws", "Storm Shield",
			"Combi-weapon", "Melta Gun", "Plasma Gun", "Flamer",
		}

		for _, weaponName := range captainWeapons {
			for _, sharedEntry := range sharedMap {
				if sharedEntry.Name == weaponName {
					for _, profile := range sharedEntry.Profiles {
						if profile.TypeName == "Weapon" || profile.TypeName == "Ranged Weapons" || profile.TypeName == "Melee Weapons" {
							if profile.Hidden != "true" {
								weaponProfile := WeaponProfile{
									Name:            profile.Name,
									Type:            profile.TypeName,
									Characteristics: make(map[string]string),
								}
								for _, char := range profile.Characteristics {
									if char.Value != "" {
										weaponProfile.Characteristics[char.Name] = char.Value
									}
								}
								if len(unit.Models) > 0 {
									if unit.Models[0].Loadouts == nil {
										unit.Models[0].Loadouts = make(map[string]WeaponProfile)
									}
									unit.Models[0].Loadouts[profile.Name] = weaponProfile
								}
							}
						}
					}
				}
			}
		}
	}

	// Special handling for units that need base weapons added to all models
	if strings.Contains(unit.Name, "Bladeguard") {
		// All Bladeguard Veterans should have Master-crafted Power Weapon
		for _, sharedEntry := range sharedMap {
			if sharedEntry.Name == "Master-crafted Power Weapon" {
				for _, profile := range sharedEntry.Profiles {
					if profile.TypeName == "Melee Weapons" && profile.Hidden != "true" {
						weaponProfile := WeaponProfile{
							Name:            profile.Name,
							Type:            profile.TypeName,
							Characteristics: make(map[string]string),
						}
						for _, char := range profile.Characteristics {
							if char.Value != "" {
								weaponProfile.Characteristics[char.Name] = char.Value
							}
						}
						// Add to all models
						for i := range unit.Models {
							if unit.Models[i].Loadouts == nil {
								unit.Models[i].Loadouts = make(map[string]WeaponProfile)
							}
							unit.Models[i].Loadouts[profile.Name] = weaponProfile
						}
					}
				}
			}
		}
	}
}

func buildSharedEntryMap(catalog *Catalogue) map[string]*SelectionEntry {
	sharedMap := make(map[string]*SelectionEntry)

	// Add shared selection entries
	for i := range catalog.SharedSelectionEntries {
		entry := &catalog.SharedSelectionEntries[i]
		sharedMap[entry.ID] = entry
		// Recursively add nested entries
		addNestedEntries(entry, sharedMap)
	}

	// Also add regular selection entries
	for i := range catalog.SelectionEntries {
		entry := &catalog.SelectionEntries[i]
		sharedMap[entry.ID] = entry
		// Recursively add nested entries
		addNestedEntries(entry, sharedMap)
	}

	return sharedMap
}

func addNestedEntries(entry *SelectionEntry, sharedMap map[string]*SelectionEntry) {
	// Add nested selection entries
	for i := range entry.SelectionEntries {
		nested := &entry.SelectionEntries[i]
		sharedMap[nested.ID] = nested
		// Recursively process deeper nesting
		addNestedEntries(nested, sharedMap)
	}

	// Add entries from selection entry groups
	for _, group := range entry.SelectionEntryGroups {
		for i := range group.SelectionEntries {
			nested := &group.SelectionEntries[i]
			sharedMap[nested.ID] = nested
			// Recursively process deeper nesting
			addNestedEntries(nested, sharedMap)
		}
	}
}

func writeUnitYAML(unit *UnitData, filename string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(unit)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

func combineUnitFiles(unit1Path, unit2Path, outputPath string) error {
	// Read and parse first unit
	unit1Data, err := ioutil.ReadFile(filepath.Join("library", unit1Path))
	if err != nil {
		return fmt.Errorf("error reading %s: %v", unit1Path, err)
	}

	var unit1 UnitData
	err = yaml.Unmarshal(unit1Data, &unit1)
	if err != nil {
		return fmt.Errorf("error parsing %s: %v", unit1Path, err)
	}

	// Read and parse second unit
	unit2Data, err := ioutil.ReadFile(filepath.Join("library", unit2Path))
	if err != nil {
		return fmt.Errorf("error reading %s: %v", unit2Path, err)
	}

	var unit2 UnitData
	err = yaml.Unmarshal(unit2Data, &unit2)
	if err != nil {
		return fmt.Errorf("error parsing %s: %v", unit2Path, err)
	}

	// Create combined unit
	combined := UnitData{
		Name: unit1.Name + " + " + unit2.Name,
		Type: "combined",
		Cost: unit1.Cost + unit2.Cost,
	}

	// Combine abilities (deduplicated)
	abilityMap := make(map[string]bool)
	for _, ability := range unit1.Abilities {
		abilityMap[ability] = true
	}
	for _, ability := range unit2.Abilities {
		abilityMap[ability] = true
	}
	combined.Abilities = []string{}
	for ability := range abilityMap {
		combined.Abilities = append(combined.Abilities, ability)
	}
	sort.Strings(combined.Abilities)

	// Combine keywords (deduplicated)
	keywordMap := make(map[string]bool)
	for _, keyword := range unit1.Keywords {
		keywordMap[keyword] = true
	}
	for _, keyword := range unit2.Keywords {
		keywordMap[keyword] = true
	}
	combined.Keywords = []string{}
	for keyword := range keywordMap {
		combined.Keywords = append(combined.Keywords, keyword)
	}
	sort.Strings(combined.Keywords)

	// Combine models
	combined.Models = append(combined.Models, unit1.Models...)
	combined.Models = append(combined.Models, unit2.Models...)

	// Combine loadout options
	combined.LoadoutOptions = append(combined.LoadoutOptions, unit1.LoadoutOptions...)
	combined.LoadoutOptions = append(combined.LoadoutOptions, unit2.LoadoutOptions...)

	// Write combined unit to output file
	outputFile := filepath.Join("library", outputPath)
	err = writeUnitYAML(&combined, outputFile)
	if err != nil {
		return fmt.Errorf("error writing combined unit: %v", err)
	}

	fmt.Printf("Combined units written to: %s\n", outputFile)
	fmt.Printf("\nCombined Unit Summary:\n")
	fmt.Printf("  Name: %s\n", combined.Name)
	fmt.Printf("  Type: %s\n", combined.Type)
	fmt.Printf("  Cost: %d\n", combined.Cost)
	fmt.Printf("  Models: %d\n", len(combined.Models))
	fmt.Printf("  Loadout Options: %d\n", len(combined.LoadoutOptions))
	fmt.Printf("  Abilities: %d\n", len(combined.Abilities))
	fmt.Printf("  Keywords: %d\n", len(combined.Keywords))

	return nil
}
