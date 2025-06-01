# BattleScribe Library Builder

A comprehensive Go-based tool for extracting and combining Warhammer 40k unit data from BattleScribe XML catalog files. Converts complex XML data into clean, simulation-ready YAML format with enhanced weapon extraction and unit combination capabilities.

## Features

- **📊 Complete Unit Data Extraction**: Stats, weapons, abilities, keywords, and costs
- **⚔️ Enhanced Weapon Processing**: Extracts ALL weapon options including composite loadouts
- **🛡️ Numerical Ability Values**: "Invulnerable Save (4+)" instead of just "Invulnerable Save"
- **📈 ISV Statistics**: Adds Invulnerable Save values directly to model stats
- **🔗 Unit Combination**: Merge multiple units for attached characters and combined forces
- **🎯 Flattened Data Structure**: Easy-to-use format for game simulation
- **📋 Individual Model Loadouts**: Weapons attached to specific models, not global profiles

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

### Extract Individual Units

```bash
go run main.go --unit "Unit Name"
```

**Examples:**
```bash
go run main.go --unit "Captain with Jump Pack"
go run main.go --unit "Bladeguard Veteran Squad"
go run main.go --unit "Brutalis Dreadnought"
```

### Combine Units

Perfect for creating attached character combinations or combined forces:

```bash
go run main.go --combine input1.yaml input2.yaml output.yaml
```

**Examples:**
```bash
# Attach a Captain to Bladeguard Veterans
go run main.go --combine captain_with_jump_pack.yaml bladeguard_veteran_squad.yaml captain_with_bladeguard.yaml

# Create a heavy support force
go run main.go --combine brutalis_dreadnought.yaml bladeguard_veteran_squad.yaml armored_escort.yaml
```

### Get Help

```bash
go run main.go --help
```

## Enhanced Output Format

The tool generates comprehensive YAML files with the following structure:

```yaml
name: Captain with Jump Pack
type: model
cost: 85
abilities:
- Angel's Wrath
- Deep Strike
- Invulnerable Save (4+)  # ← Enhanced with numerical values
- Leader
- Oath of Moment
- Rites of Battle
keywords:
- Captain
- Character
- 'Faction: Adeptus Astartes'
- Fly
- Infantry
- Jump Pack
models:
- name: Captain with Jump Pack
  count: 1
  stats:
    ISV: 4+      # ← Invulnerable Save in stats
    LD: 6+
    M: 12"
    OC: "1"
    SV: 3+
    T: "4"
    W: "5"
  loadouts:     # ← Weapons attached to individual models
    Bolt Pistol:
      name: Bolt Pistol
      type: Ranged Weapons
      A: "1"
      AP: "0"
      BS: 3+
      D: "1"
      Keywords: Pistol
      Range: 12"
      S: "4"
    Master-crafted Power Weapon:
      name: Master-crafted Power Weapon
      type: Melee Weapons
      A: "4"
      AP: "-2"
      D: "2"
      Keywords: Precision
      Range: Melee
      S: "5"
      WS: 2+
    # ... 11 more weapons extracted!
loadout_options:
- name: Wargear
  type: group
  options:
  - Melee and Pistol
  - Thunder Hammer and Relic Shield
```

### Key Improvements

#### 🎯 **Enhanced Weapon Extraction**
- **Before**: Captain with Jump Pack had 1 weapon
- **After**: Captain with Jump Pack has 13 weapons including all pistol and melee options
- Processes composite loadouts like "Melee and Pistol"
- Extracts weapons from nested XML structures and entry links

#### 🛡️ **Ability Values**
- **Before**: "Invulnerable Save"
- **After**: "Invulnerable Save (4+)"
- Supports all save values: 2+, 3+, 4+, 5+, 6+
- Extracts from both local rules and shared profiles

#### 📊 **ISV in Stats**
```yaml
stats:
  ISV: 4+      # ← Only appears if unit has invulnerable save
  LD: 6+
  M: 6"
  OC: "1"
  SV: 3+       # ← Regular armor save
  T: "4"
  W: "3"
```

#### 🔗 **Unit Combination**
Combined units include:
- **Concatenated names**: "Captain with Jump Pack + Bladeguard Veteran Squad"
- **Added costs**: 85 + 80 = 165 points
- **All models**: Preserves individual model loadouts and stats
- **Merged abilities**: Deduplicated list of all abilities
- **Combined keywords**: Unified keyword list
- **All loadout options**: Equipment choices from both units

## Example Workflow

### 1. Extract Individual Units
```bash
# Extract a character
go run main.go --unit "Captain with Jump Pack"

# Extract a squad
go run main.go --unit "Bladeguard Veteran Squad"

# Extract heavy support
go run main.go --unit "Brutalis Dreadnought"
```

### 2. Combine for Attached Characters
```bash
# Create attached character unit
go run main.go --combine captain_with_jump_pack.yaml bladeguard_veteran_squad.yaml elite_strike_team.yaml
```

### 3. Use in Simulation
The resulting YAML files are ready for immediate use in game simulation with:
- Complete weapon profiles with all characteristics
- Individual model stats including ISV
- Comprehensive ability and keyword lists
- Clear loadout options for customization

## Enhanced Data Structure

### Model-Specific Loadouts
Unlike the previous global weapon profiles, weapons are now attached to individual models:

```yaml
models:
- name: Bladeguard Veteran        # Regular veteran
  loadouts:
    Heavy Bolt Pistol: {...}
    Master-crafted Power Weapon: {...}
    
- name: Bladeguard Veteran Sergeant   # Sergeant with more options
  loadouts:
    Heavy Bolt Pistol: {...}
    Neo-volkite Pistol: {...}
    "➤ Plasma Pistol - Standard": {...}
    "➤ Plasma Pistol - Supercharge": {...}
    Master-crafted Power Weapon: {...}
```

### Flattened Weapon Characteristics
All weapon stats are at the top level for easy access:

```yaml
Thunder Hammer:
  name: Thunder Hammer
  type: Melee Weapons
  A: "3"           # Direct access to all stats
  AP: "-2"
  D: "2"
  Keywords: Devastating Wounds
  Range: Melee
  S: "8"
  WS: 4+
```

## Finding Unit Names

Unit names must match exactly. Search the catalog files:

**Windows:**
```cmd
findstr /i "unit_name" battlescribe-data-10e\*.cat
```

**Linux/Mac:**
```bash
grep -i "unit_name" battlescribe-data-10e/*.cat
```

## Troubleshooting

### Unit Not Found
1. Check exact spelling and capitalization
2. Include full unit name (e.g., "Bladeguard Veteran Squad")
3. Search catalog files for exact name

### Missing Weapons
- The tool now extracts composite loadouts automatically
- If weapons are still missing, they may be in deeply nested XML structures
- Check the unit's loadout_options for available choices

### File Permissions
- Ensure `library/` directory is writable
- Check BattleScribe data file permissions

## Technical Details

### Supported Features
- ✅ Complex nested XML parsing
- ✅ Shared entry and profile resolution
- ✅ Recursive weapon extraction
- ✅ Composite loadout processing
- ✅ Ability value extraction
- ✅ ISV calculation and assignment
- ✅ Unit combination with deduplication
- ✅ Error handling and validation

### Performance
- Processes units in seconds
- Handles complex multi-model units
- Efficient memory usage for large catalogs
- Tab-completion friendly command interface

This enhanced library builder provides comprehensive, simulation-ready Warhammer 40k unit data with all the detail needed for accurate game modeling and tactical analysis. 