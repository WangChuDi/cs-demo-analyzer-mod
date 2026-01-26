package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// Map equipment type to generic weapon name string matching keys in constants.WeaponPrices
var equipmentToWeaponName = map[common.EquipmentType]constants.WeaponName{
	common.EqSmoke:      constants.WeaponSmoke,
	common.EqFlash:      constants.WeaponFlashbang,
	common.EqHE:         constants.WeaponHEGrenade,
	common.EqIncendiary: constants.WeaponIncendiary,
	common.EqMolotov:    constants.WeaponMolotov,
	common.EqDecoy:      constants.WeaponDecoy,
}

func TestWastedUtilityLogic(t *testing.T) {
	// 1. Open demo file
	demoPath := "../../cs-demos/9210178424085284364_0.dem"
	f, err := os.Open(demoPath)
	if err != nil {
		t.Fatalf("failed to open demo file: %v", err)
	}
	defer f.Close()

	// 2. Create parser
	p := demoinfocs.NewParser(f)
	defer p.Close()

	// 3. Register event handler to calculate wasted utility
	totalWastedValue := 0

	p.RegisterEventHandler(func(e events.Kill) {
		if e.Victim != nil {
			for _, weapon := range e.Victim.Weapons() {
				if weapon == nil {
					continue
				}

				// Only care about grenades
				if weapon.Class() != common.EqClassGrenade {
					continue
				}

				// Map to API constant name
				name, ok := equipmentToWeaponName[weapon.Type]
				if !ok {
					continue
				}

				// Get price from API constants
				if price, ok := constants.WeaponPrices[name]; ok {
					totalWastedValue += price
					fmt.Printf("Round %d: Player %s died with %s (Value: $%d)\n", p.GameState().TeamCounterTerrorists().Score()+p.GameState().TeamTerrorists().Score()+1, e.Victim.Name, name, price)
				}
			}
		}
	})

	// 4. Parse demo
	err = p.ParseToEnd()
	if err != nil {
		t.Fatalf("failed to parse demo: %v", err)
	}

	fmt.Printf("Total Wasted Utility Value: $%d\n", totalWastedValue)

	// Basic assertion to verify we found something (value depends on the specific demo)
	if totalWastedValue == 0 {
		t.Error("Expected total wasted utility value to be greater than 0")
	}
}
