package main

import (
	"testing"
)

func TestLoadUnit(t *testing.T) {
	defer func() { recover() }()
	_ = loadUnit("bladeguard_veterans.yaml")
}

func TestUnit_PrintInfo(t *testing.T) {
	u := loadUnit("bladeguard_veterans.yaml")
	u.PrintInfo()
}

func TestUnit_calcToughness(t *testing.T) {
	u := loadUnit("bladeguard_veterans.yaml")
	_ = u.calcToughness()
}

func TestUnit_removeAbility(t *testing.T) {
	u := loadUnit("bladeguard_veterans.yaml")
	_ = u.removeAbility("BLADEGUARD")
}

func TestUnitAttackSequence_applyDamage(t *testing.T) {
	attacker := loadUnit("bladeguard_veterans.yaml")
	defender := loadUnit("bladeguard_veterans.yaml")
	conflict := UnitAttackSequence{Attacker: attacker, Defender: defender}
	modelName := defender.ModelOrder[0]
	conflict.applyDamage(modelName, "2")
}

func TestUnit_Reload(t *testing.T) {
	u := loadUnit("bladeguard_veterans.yaml")
	u.Reload()
}

func TestUnit_Reset(t *testing.T) {
	u := loadUnit("bladeguard_veterans.yaml")
	u.Reset()
}
