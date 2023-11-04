package main

import (
	"io/ioutil"
	"sort"

	"gopkg.in/yaml.v2"
)

const _unitLibraryFilepath = "../library"

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
	Type             string `yaml:"type"`
	Name             string
	Range            int      `yaml:"range,omitempty"`
	N                string   `yaml:"N"`
	D                string   `yaml:"D"`
	Skill            int      `yaml:"skill"`
	Ap               int      `yaml:"Ap"`
	LoudoutAbilities []string `yaml:"abilities,omitempty"`

	Modifiers struct {
		RerollHits    bool
		RerollHit1s   bool
		RerollWounds  bool
		RerollWound1s bool
		HitMod        int
		WoundMod      int
		CritHit       int
		CritWound     int
		CritHitFish   bool
		CritWoundFish bool
	}
}

func loadUnit(name string) Unit {
	var (
		data []byte
		err  error
	)

	if data, err = ioutil.ReadFile(_unitLibraryFilepath + name); err != nil {
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
