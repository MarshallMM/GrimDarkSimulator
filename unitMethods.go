package main

import (
	"io/ioutil"

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
	return unit
}
