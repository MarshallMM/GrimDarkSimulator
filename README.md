# GrimDarkSimulator

A comprehensive Warhammer 40k combat simulator written in Go that accurately models the complete combat sequence with detailed statistical analysis and granular combat logging.

## Features

### 🎯 Complete Combat Resolution
- **Hit Phase**: Skill-based hit rolls (BS/WS) with modifier support
- **Wound Phase**: Strength vs Toughness matrix calculations
- **Save Phase**: Dual save system (Armor/Invulnerable) with AP modification
- **Damage Phase**: Variable damage application with model removal

### ⚔️ Advanced Combat Mechanics
- **Critical Hits**: Rolls ≥ CritHit modifier trigger special effects
- **Lethal Hits**: Critical hits auto-wound (bypass wound rolls)
- **Sustained Hits X**: Critical hits generate X additional hits
- **Devastating Wounds**: Critical wound rolls bypass all saves
- **Torrent Weapons**: Auto-hit weapons skip hit rolls entirely

### 🔄 Reroll Systems
- **Hit Rerolls**: Full rerolls or reroll 1s only
- **Wound Rerolls**: Full rerolls or reroll 1s only
- **Complete Transparency**: All original and reroll results logged

### 🛡️ Abilities System
The simulator automatically detects and applies unit abilities that modify combat characteristics:

#### Unit Abilities
- **Oath of Moment**: All weapons gain reroll hits capability
- **Stationary**: Heavy weapons receive +1 to hit modifier
- **Rapid Fire Distance**: Rapid Fire weapons gain additional attacks equal to their Rapid Fire value
- **Red Rampage**: Melee weapons gain both Lethal Hits and Lance abilities, but unit becomes Battle-shocked

#### Weapon Abilities
- **Twin-linked**: Weapons with "Twin-linked" in name or keywords gain reroll wounds
- **Heavy**: When combined with "Stationary" ability, gains +1 to hit
- **Rapid Fire X**: When combined with "Rapid Fire Distance" ability, gains X additional attacks

### 📊 Statistical Analysis
- **100 Simulation Runs**: Comprehensive damage distribution
- **Percentile Analysis**: 68th and 95th percentile damage outputs
- **Realistic Results**: Balanced outcomes reflecting tabletop play

### 📝 Granular Combat Logging
- **Phase-by-Phase Breakdown**: Every step of combat logged
- **Individual Dice Rolls**: All hit/wound/save rolls recorded
- **Special Rule Tracking**: Critical hits, rerolls, and abilities logged
- **Ability Applications**: All ability modifications logged with before/after values
- **Defender Status**: Remaining models and wounds after each attack
- **JSON Structured Logs**: Machine-readable combat data

## Quick Start

### Prerequisites
- Go 1.21 or later
- Unit files in YAML format (from library_builder)

### Installation
```bash
git clone <repository-url>
cd GrimDarkSimulator
go mod tidy
```

### Running Simulations
```bash
go run .
```

## Unit File Format

Units are loaded from the `./library/` directory in YAML format:

```yaml
name: "Captain with Jump Pack"
type: "Character"
cost: 85
abilities:
  - "Leader"
  - "Deep Strike"
  - "Oath of Moment"  # Grants reroll hits to all weapons
keywords:
  - "Infantry"
  - "Character"
models:
  - name: "Captain with Jump Pack"
    count: 1
    stats:
      M: "12\""
      T: "4"
      SV: "3+"
      W: "5"
      LD: "6+"
      OC: "1"
    loadouts:
      "Master-crafted Power Weapon":
        name: "Master-crafted power weapon"
        type: "Melee Weapons"
        A: "5"
        WS: "2+"
        S: "5"
        AP: "2"
        D: "2"
        Keywords: "Lethal Hits, Sustained Hits 1"
      "Heavy Bolter":
        name: "Heavy Bolter"
        type: "Ranged Weapons"
        Range: "36\""
        A: "3"
        BS: "3+"
        S: "5"
        AP: "1"
        D: "2"
        Keywords: "Heavy, Sustained Hits 1"  # Benefits from Stationary
```

## Combat Mechanics

### Hit Resolution
1. **Abilities Check**: Apply unit and weapon abilities that modify hit characteristics
2. **Skill Check**: Roll vs BS (ranged) or WS (melee)
3. **Critical Hits**: Rolls ≥ CritHit value (default 6+)
4. **Special Rules**: 
   - Lethal Hits: Critical hits auto-wound
   - Sustained Hits X: Add X hits per critical
5. **Rerolls**: Apply hit rerolls if available

### Wound Resolution  
1. **S vs T Matrix**: Calculate wound threshold
   - S ≥ 2×T: 2+ to wound
   - S > T: 3+ to wound  
   - S = T: 4+ to wound
   - S < T: 5+ to wound
   - S ≤ T/2: 6+ to wound
2. **Devastating Wounds**: Critical wounds (6+ by default) bypass saves
3. **Lethal Hits**: Auto-wound without rolling
4. **Rerolls**: Apply wound rerolls if available

### Save Resolution
1. **Devastating Wounds**: Apply damage directly (no saves)
2. **Save Priority**: Invulnerable saves checked first
3. **AP Modification**: Armor saves modified by weapon AP
4. **Damage Application**: Failed saves apply weapon damage

### Abilities Processing
Abilities are automatically applied at the start of each combat sequence:

1. **Unit Abilities**: Processed first, affecting all applicable weapons
2. **Weapon Keywords**: Processed second, affecting specific weapons
3. **Logging**: All ability applications are logged with before/after values
4. **Stacking**: Multiple modifiers can stack (e.g., +1 hit from multiple sources)

## Abilities Reference

### Oath of Moment
- **Effect**: All weapons gain `RerollHits = true`
- **Application**: Applied to every weapon in every model's loadout
- **Usage**: Represents focused targeting and battle prayer benefits

### Stationary
- **Effect**: Heavy weapons gain `HitMod += 1` (+1 to hit)
- **Application**: Only affects weapons with "Heavy" keyword
- **Usage**: Represents the accuracy benefit of stationary firing positions

### Rapid Fire Distance
- **Effect**: Rapid Fire weapons gain additional attacks equal to their Rapid Fire value
- **Application**: Parses "Rapid Fire X" keywords and adds X to the weapon's attack count
- **Usage**: Represents double-tapping at close range

### Twin-linked
- **Effect**: Weapons gain `RerollWounds = true`
- **Application**: Affects weapons with "Twin-linked" in name or keywords
- **Usage**: Represents multiple barrels/enhanced targeting systems

## Combat Log Example

```
Applied Oath of Moment: All weapons gain reroll hits
Applied Stationary: Heavy weapons gain +1 to hit
Applied Rapid Fire Distance: Rapid Fire Lasgun gains +1 attacks (2 → 3)
Applied Twin-linked: Twin-linked Autocannon gains reroll wounds

Starting weapon attack: Captain attacking with Master-crafted Power Weapon
Hit Phase - Rolling: 5 attacks, need 2+ to hit
  Hit Roll 1: Rolled 6, critical hit, hit
  Hit Roll 2: Rolled 3, normal hit  
  Hit Roll 3: Rolled 1, miss, rerolling due to Oath of Moment
  Hit Reroll 3: Rolled 4, hit
Hit Phase Complete: 3 hits (0 Sustained Hits, 0 Lethal Hits)

Wound Phase: S5 vs T4, need 3+ to wound
  Wound Roll 1: Rolled 6, Devastating Wound
  Wound Roll 2: Rolled 4, normal wound
Wound Phase Complete: 2 wounds (1 Devastating Wounds)

Save Phase: SV 3+ (modified by AP-2), ISV 4+
  Devastating Wound 1: Bypasses save, damage applied
  Save Roll 2: Rolled 3, failed save, damage applied
Save Phase Complete: 2 damage (1 from Devastating Wounds, 1 from failed saves)

Weapon Attack Complete: 2 damage applied, 1 model remaining
```

## Project Structure

```
GrimDarkSimulator/
├── main.go              # Entry point and simulation control
├── unitMethods.go       # Core combat system and unit handling  
├── util.go             # Utility functions (dice rolling, etc.)
├── library/            # Unit YAML files
├── library_builder/    # BattleScribe XML to YAML converter
├── combat_log.txt      # Detailed combat logs (first simulation only)
└── README.md          # This file
```

## Configuration

Weapon modifiers can be configured in the unit YAML:

```yaml
# Weapon modifiers (set during unit loading)
Modifiers:
  RerollHits: false      # Reroll all failed hits
  RerollHit1s: false     # Reroll hit rolls of 1  
  RerollWounds: false    # Reroll all failed wounds
  RerollWound1s: false   # Reroll wound rolls of 1
  HitMod: 0             # Modifier to hit rolls (+/-)
  WoundMod: 0           # Modifier to wound rolls (+/-)
  CritHit: 6            # Critical hit threshold
  CritWound: 6          # Critical wound threshold
```

## Special Rules Implementation

- **Torrent**: Weapons auto-hit (skip hit phase)
- **Lethal Hits**: Critical hits automatically wound
- **Sustained Hits X**: Critical hits generate X additional hits  
- **Devastating Wounds**: Critical wounds bypass all saves
- **Heavy**: Benefits from Stationary ability (+1 to hit)
- **Rapid Fire X**: Benefits from Rapid Fire Distance ability (+X attacks)
- **Twin-linked**: Weapons gain reroll wounds capability
- **Feel No Pain**: Post-save damage reduction (if detected in abilities)
- **NECRODERMIS**: Halves incoming damage (special faction rule)

## Dependencies

- `go.uber.org/zap`: Structured logging
- `gopkg.in/yaml.v2`: YAML parsing

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add comprehensive tests for new combat mechanics
4. Ensure all simulations produce balanced results
5. Submit a pull request

## License

This project is for educational and simulation purposes. Warhammer 40,000 is a trademark of Games Workshop Ltd.
