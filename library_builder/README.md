# BattleScribe Library Builder

A Go-based tool for extracting Warhammer 40k unit data from BattleScribe XML catalog files and converting them into clean YAML format for use in game simulators and other applications.

## Overview

This tool parses BattleScribe XML catalog files (`.cat` format) and extracts detailed unit information including:

- **Unit Stats**: Movement, Toughness, Save, Leadership, etc.
- **Weapon Profiles**: Range, Attacks, Ballistic/Weapon Skill, Strength, AP, Damage, Keywords
- **Loadout Options**: Available weapon and equipment choices
- **Unit Abilities**: Special rules and abilities
- **Keywords**: Faction keywords, unit types, etc.
- **Point Costs**: Current competitive point values

## Prerequisites

- **Go**: Version 1.16 or later
- **BattleScribe Data**: The official BattleScribe data repository

## Setup

1. **Clone or download the BattleScribe data repository:**
   ```bash
   git clone https://github.com/BSData/wh40k-10e.git battlescribe-data-10e
   ```

2. **Ensure your directory structure looks like this:**
   ```
   library_builder/
   ├── main.go
   ├── battlescribe-data-10e/
   │   ├── Imperium - Space Marines.cat
   │   ├── Chaos - Chaos Space Marines.cat
   │   └── ... (other catalog files)
   └── library/
       └── (generated YAML files will go here)
   ```

## Usage

### Basic Command

```bash
go run main.go --unit "Unit Name"
```

### Examples

Extract a Brutalis Dreadnought:
```bash
go run main.go --unit "Brutalis Dreadnought"
```

Extract an Intercessor Squad:
```bash
go run main.go --unit "Intercessor Squad"
```

Extract a Chaos unit:
```bash
go run main.go --unit "Legionaries"
```

Extract a vehicle:
```bash
go run main.go --unit "Land Raider"
```

### Command-Line Options

- `--unit "Unit Name"`: Specifies the exact name of the unit to extract (required)

## Output Format

The tool generates YAML files in the `library/` directory with the following structure:

```yaml
name: Brutalis Dreadnought
type: model
cost: 160
abilities:
- 'Damaged: 1-4 Wounds Remaining'
- Deadly Demise
- Oath of Moment
keywords:
- Brutalis Dreadnought
- Dreadnought
- 'Faction: Adeptus Astartes'
- Imperium
- Vehicle
- Walker
models:
- name: Brutalis Dreadnought
  count: 1
  stats:
    LD: 6+
    M: 8"
    OC: "4"
    SV: 2+
    T: "10"
    W: "12"
loadout_options:
- name: Melee Weapon Option
  type: group
  options:
  - Brutalis Fists & Brutalis Bolt Rifles
  - Brutalis Talons
- name: Ranged Weapon Option
  type: group
  options:
  - Twin Heavy Bolter
  - Twin Multi-melta
profiles:
  Twin Heavy Bolter:
    name: Twin Heavy Bolter
    type: Ranged Weapons
    A: "3"
    AP: "-1"
    BS: 3+
    D: "2"
    Keywords: Sustained Hits 1, Twin-linked
    Range: 36"
    S: "5"
```

### Output Fields Explained

- **name**: Unit name from BattleScribe
- **type**: Unit type (unit, model, upgrade)
- **cost**: Point cost in matched play
- **abilities**: List of special rules and abilities
- **keywords**: Faction keywords, unit types, and special designations
- **models**: Individual model entries with stats and equipment
- **loadout_options**: Available weapon and equipment choices organized by group
- **profiles**: Detailed weapon and equipment profiles with full characteristics

### Weapon Profile Format

Weapon characteristics are presented in a flat, easy-to-read format:

- **Range**: Weapon range (e.g., "36\"", "Melee")
- **A**: Number of attacks (e.g., "3", "D6")
- **BS/WS**: Ballistic/Weapon Skill (e.g., "3+", "4+")
- **S**: Strength (e.g., "5", "8")
- **AP**: Armor Penetration (e.g., "-1", "-3")
- **D**: Damage (e.g., "2", "D6")
- **Keywords**: Special weapon abilities (e.g., "Twin-linked", "Melta 2")

## Finding Unit Names

Since unit names must match exactly, you can search the catalog files to find the correct name:

### Windows (Command Prompt/PowerShell)
```cmd
findstr /i "unit_name" battlescribe-data-10e\*.cat
```

### Linux/Mac
```bash
grep -i "unit_name" battlescribe-data-10e/*.cat
```

### Common Unit Name Patterns

- Space Marine squads often end with "Squad" (e.g., "Intercessor Squad")
- Individual characters usually have no suffix (e.g., "Captain")
- Vehicles are often single names (e.g., "Land Raider", "Predator")
- Some units have specific variants (e.g., "Brutalis Dreadnought")

## Troubleshooting

### "Unit not found" Error

1. **Check the exact spelling** - Unit names are case-sensitive
2. **Include the full name** - Use "Bladeguard Veteran Squad" not "Bladeguard Veterans"
3. **Search the catalog files** to find the exact name used
4. **Try variations** - Some units might be in different catalogs or have alternate names

### Empty or Missing Data

- Some units may have incomplete data if they're in draft or beta status
- Vehicle squadrons sometimes have data spread across multiple entries
- Named characters might be in faction-specific catalogs

### Permission Errors

- Ensure the `library/` directory is writable
- Check that you have permission to read the BattleScribe data files

## Example Workflow

1. **Find your unit:**
   ```bash
   grep -i "intercessor" battlescribe-data-10e/*.cat
   ```

2. **Extract the unit:**
   ```bash
   go run main.go --unit "Intercessor Squad"
   ```

3. **Check the output:**
   ```bash
   cat library/intercessor_squad.yaml
   ```

4. **Use in your application:**
   ```go
   // Load the YAML file in your Go application
   data, err := ioutil.ReadFile("library/intercessor_squad.yaml")
   // Parse with yaml.Unmarshal()
   ```

## Contributing

To improve the tool:

1. **Add support for new data types** - Extend the XML structures
2. **Improve parsing logic** - Handle edge cases and special unit types  
3. **Add validation** - Ensure extracted data is complete and accurate
4. **Optimize performance** - Cache shared entries and improve search algorithms

## License

This tool is for personal and educational use. BattleScribe data is owned by Games Workshop and maintained by the BSData community. 