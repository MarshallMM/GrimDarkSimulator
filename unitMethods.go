package main

import (
	"fmt"
	"math"
	"os"
	"sort"

	"gopkg.in/yaml.v2"
)

const _unitLibraryFilepath = "./library/"
const _heavyComments = true

type UnitAttackSequence struct {
	Attacker Unit
	Defender Unit
}
type Unit struct {
	Source        string
	Cost          int                     `yaml:"cost"`
	Models        map[string]ModelDetails `yaml:",inline"`
	UnitAbilities []string                `yaml:"UnitAbilities,omitempty"`
	ModelOrder    []string
}

type ModelDetails struct {
	Count           int `yaml:"count"`
	Priority        int `yaml:"priority"`
	Killed          int
	Wounds          int `yaml:"wounds"`
	CarryOverWounds int
	Keywords        []string `yaml:"keywords,omitempty"`
	Abilities       []string `yaml:"abilities,omitempty"`

	T        int `yaml:"T"`
	Sv       int `yaml:"Sv"`
	ISv      int `yaml:"iSv,omitempty"`
	FNP      int `yaml:"FNP,omitempty"`
	MWFNP    int
	Loadouts map[string]LoadoutData `yaml:"loadouts"`
}

type LoadoutData struct {
	Type      string   `yaml:"type"`
	Name      string   `yaml:"name"`
	Range     int      `yaml:"range,omitempty"`
	N         string   `yaml:"N"`
	D         string   `yaml:"D"`
	A         int      `yaml:"A,omitempty"`
	S         int      `yaml:"S,omitempty"`
	Ap        int      `yaml:"Ap,omitempty"`
	Abilities []string `yaml:"abilities,omitempty"`
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
	// sort priorities
	m := unit.Models
	pairs := make([][2]interface{}, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, [2]interface{}{k, v.Priority})
	}

	// sort slice based on values
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i][1].(int) < pairs[j][1].(int)
	})

	// extract sorted pairs
	keys := make([]string, len(pairs))
	for i, p := range pairs {
		keys[i] = p[0].(string)
	}
	unit.ModelOrder = keys
	unit.Source = name
	for modelName, model := range unit.Models {
		unitData := unit.Models[modelName]
		unitData.Killed = 0
		unitData.CarryOverWounds = 0

		for loadoutName, loadOut := range model.Loadouts {
			loadOut.Modifiers.CritHit = 6
			loadOut.Modifiers.CritHitFish = false
			loadOut.Modifiers.CritWound = 6
			loadOut.Modifiers.CritWoundFish = false
			loadOut.Modifiers.HitMod = 0
			loadOut.Modifiers.WoundMod = 0
			loadOut.Modifiers.RerollHit1s = false
			loadOut.Modifiers.RerollHits = false
			loadOut.Modifiers.RerollWound1s = false
			loadOut.Modifiers.RerollWounds = false
			unit.Models[modelName].Loadouts[loadoutName] = loadOut
		}
	}
	return unit
}

func (u *Unit) PrintInfo() {
	if _heavyComments {
		for modelName, unitData := range u.Models {
			fmt.Printf("Model: %s | Killed: %v | Wounds: %v \n", modelName, unitData.Killed, unitData.CarryOverWounds)
			for loadoutname, loudoutdata := range unitData.Loadouts {
				fmt.Printf("Loadout: %s | %+v\n", loadoutname, loudoutdata)
			}
		}
	}
}

func (u *Unit) calcToughness() int {
	toughness := 0
	// iterate through bodgy guard unit and get highest toughness
	for _, modelName := range u.ModelOrder {
		DefunitData := u.Models[modelName]
		if DefunitData.Count > DefunitData.Killed {
			if exists, _ := stringExistsInSlice("LEADER", DefunitData.Abilities); !exists {
				if toughness < DefunitData.T {
					toughness = DefunitData.T
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
	return used
}

func (conflict *UnitAttackSequence) applyDamage(modelName string, damString string, params ...string) {
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

	defData := conflict.Defender.Models[modelName]

	if defData.FNP > 0 || (mortals && defData.MWFNP > 0) {
		threashold := 0
		if defData.FNP > threashold {
			threashold = defData.FNP
		}
		if mortals && defData.MWFNP > threashold {
			threashold = defData.MWFNP
		}

		initialDamage := damage
		for i := 0; i < initialDamage; i++ {
			roll := rollDice(1, 6)
			if roll >= threashold {
				damage = damage - 1
			}
		}
	}
	if abilityCheck("NECRODERMIS", conflict.Defender.UnitAbilities) {
		damage = int(math.Ceil(float64(damage) / 2))
	}
	defData.CarryOverWounds = defData.CarryOverWounds + damage
	if defData.CarryOverWounds >= defData.Wounds && defData.Killed < defData.Count {
		defData.Killed++
		defData.CarryOverWounds = 0
	}
	conflict.Defender.Models[modelName] = defData
}
func (u *Unit) Reload() {
	resetUnit := loadUnit(u.Source)
	u.UnitAbilities = resetUnit.UnitAbilities
	for modelName := range resetUnit.Models {
		unitData := u.Models[modelName]
		unitData.CarryOverWounds = 0
		u.Models[modelName] = unitData
		for loadoutName := range u.Models[modelName].Loadouts {
			u.Models[modelName].Loadouts[loadoutName] = resetUnit.Models[modelName].Loadouts[loadoutName]
		}
	}
	u.Reset()
}
func (u *Unit) Reset() {
	for modelName, model := range u.Models {
		unitData := u.Models[modelName]
		u.Models[modelName] = unitData
		for loadOutName, loadOut := range model.Loadouts {
			loadOut.Modifiers.CritHit = 6
			loadOut.Modifiers.CritHitFish = false
			loadOut.Modifiers.CritWound = 6
			loadOut.Modifiers.CritWoundFish = false
			loadOut.Modifiers.HitMod = 0
			loadOut.Modifiers.WoundMod = 0
			loadOut.Modifiers.RerollHit1s = false
			loadOut.Modifiers.RerollHits = false
			loadOut.Modifiers.RerollWound1s = false
			loadOut.Modifiers.RerollWounds = false
			u.Models[modelName].Loadouts[loadOutName] = loadOut
		}
	}
}
