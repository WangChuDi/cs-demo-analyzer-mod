package api

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

func utilityImpactLifecycleFixture() (*Analyzer, *Damage, *Kill) {
	analyzer, kill := utilityKillFixture()
	kill.setUtilityImpactData("hegrenade")
	analyzer.match.Kills = append(analyzer.match.Kills, kill)
	damage := &Damage{Frame: kill.Frame, Tick: kill.Tick, RoundNumber: 1,
		AttackerSteamID64: kill.KillerSteamID64, VictimSteamID64: kill.VictimSteamID64,
		AttackerSide: kill.KillerSide, VictimSide: kill.VictimSide,
		WeaponName: constants.WeaponHEGrenade, WeaponUniqueID: "7", HealthDamage: 1, VictimHealth: 1, VictimNewHealth: 0,
		utilityImpactCandidates: []utilityImpactCandidate{{id: 42, ownerMatches: true}}}
	analyzer.match.Damages = []*Damage{damage}
	analyzer.utilityImpactLifetimes = map[common.UniqueID]*utilityImpactLifetime{42: {
		id: 42, entityID: 10, weaponID: "7", kind: common.EqHE, round: 1,
		hasThrow: true, throwTick: 100, throwFrame: 100,
		hasEffect: true, effectTick: 228, effectFrame: 228, effect: "he_explode",
		hasDestroy: true, destroyTick: 548, destroyFrame: 548,
	}}
	return analyzer, damage, kill
}

func TestUtilityImpactLifecycleFatalHEAndDecoy(t *testing.T) {
	for _, decoy := range []bool{false, true} {
		analyzer, damage, kill := utilityImpactLifecycleFixture()
		if decoy {
			kill.WeaponName, damage.WeaponName = constants.WeaponDecoy, constants.WeaponDecoy
			kill.setUtilityImpactData("decoy")
			analyzer.utilityImpactLifetimes[42].kind = common.EqDecoy
			analyzer.utilityImpactLifetimes[42].effect = "decoy_expired"
		}
		analyzer.resolveUtilityImpactLifecycles()
		if !kill.HasUtilityImpactData || !kill.IsUtilityImpactKill || kill.UtilityImpactSource != utilityImpactSourceLifecycle ||
			kill.UtilityImpactProjectileID != "42" || kill.UtilityImpactEffectTick != 228 ||
			kill.UtilityImpactEffect != analyzer.utilityImpactLifetimes[42].effect ||
			(&Player{SteamID64: 3, match: analyzer.match}).UtilityImpactDeathCount() != 1 {
			t.Fatalf("missing fatal lifecycle evidence: %+v", kill)
		}
		// Resolution is idempotent and does not turn the original attacker death
		// or a missing generic death event into another kill.
		analyzer.resolveUtilityImpactLifecycles()
		if (&Player{SteamID64: 3, match: analyzer.match}).UtilityImpactDeathCount() != 1 {
			t.Fatal("repeated resolution changed the total")
		}
	}
}

func TestUtilityImpactLifecycleRespectsRawEvidence(t *testing.T) {
	for _, raw := range []string{"", "ak47", "smokegrenade"} {
		t.Run(raw, func(t *testing.T) {
			analyzer, _, kill := utilityImpactLifecycleFixture()
			kill.setUtilityImpactData(raw)
			analyzer.resolveUtilityImpactLifecycles()
			if kill.RawWeaponName != raw || !kill.HasUtilityImpactData {
				t.Fatal("raw event was lost or missing raw data blocked lifecycle proof")
			}
			if raw == "" {
				if !kill.IsUtilityImpactKill || kill.UtilityImpactSource != utilityImpactSourceLifecycle {
					t.Fatal("missing raw event did not fall back to lifecycle proof")
				}
			} else if kill.UtilityImpactSource != utilityImpactSourceRawWeapon || kill.UtilityImpactProjectileID != "" ||
				kill.IsUtilityImpactKill != (raw == "smokegrenade") {
				t.Fatal("lifecycle fallback overwrote an explicit raw cause")
			}
		})
	}
}

func TestUtilityImpactLifecycleRejectsUncertainEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Analyzer, *Damage, *Kill)
	}{
		{"effect missing", func(a *Analyzer, _ *Damage, _ *Kill) { a.utilityImpactLifetimes[42].hasEffect = false }},
		{"throw missing", func(a *Analyzer, _ *Damage, _ *Kill) { a.utilityImpactLifetimes[42].hasThrow = false }},
		{"lifetime missing", func(a *Analyzer, _ *Damage, _ *Kill) { delete(a.utilityImpactLifetimes, 42) }},
		{"unknown owner", func(_ *Analyzer, d *Damage, _ *Kill) { d.utilityImpactCandidates[0].ownerMatches = false }},
		{"round mismatch", func(a *Analyzer, _ *Damage, _ *Kill) { a.utilityImpactLifetimes[42].round++ }},
		{"same tick later frame", func(a *Analyzer, d *Damage, _ *Kill) { a.utilityImpactLifetimes[42].effectTick = d.Tick }},
		{"same frame later tick", func(a *Analyzer, d *Damage, _ *Kill) { a.utilityImpactLifetimes[42].effectFrame = d.Frame }},
		{"already exploded entity still exists", func(a *Analyzer, d *Damage, _ *Kill) {
			a.utilityImpactLifetimes[42].effectTick, a.utilityImpactLifetimes[42].effectFrame = d.Tick-1, d.Frame-1
		}},
		{"effect before throw", func(a *Analyzer, _ *Damage, _ *Kill) { a.utilityImpactLifetimes[42].effectTick = 99 }},
		{"effect after cleanup", func(a *Analyzer, _ *Damage, _ *Kill) { a.utilityImpactLifetimes[42].destroyTick = 200 }},
		{"throw after hurt", func(a *Analyzer, d *Damage, _ *Kill) { a.utilityImpactLifetimes[42].throwFrame = d.Frame + 1 }},
		{"ambiguous effects", func(a *Analyzer, _ *Damage, _ *Kill) { a.utilityImpactLifetimes[42].ambiguous = true }},
		{"wrong weapon instance", func(_ *Analyzer, d *Damage, _ *Kill) { d.WeaponUniqueID = "8" }},
		{"wrong grenade type", func(a *Analyzer, _ *Damage, _ *Kill) { a.utilityImpactLifetimes[42].kind = common.EqDecoy }},
		{"no projectile candidates", func(_ *Analyzer, d *Damage, _ *Kill) { d.utilityImpactCandidates = nil }},
		{"attacker unknown", func(_ *Analyzer, d *Damage, _ *Kill) { d.AttackerSteamID64 = 0 }},
		{"controlling bot", func(_ *Analyzer, d *Damage, _ *Kill) { d.IsAttackerControllingBot = true }},
		{"multiple live projectiles", func(a *Analyzer, d *Damage, _ *Kill) {
			other := *a.utilityImpactLifetimes[42]
			other.id = 43 // Same inventory weapon ID; a different throw.
			a.utilityImpactLifetimes[43] = &other
			d.utilityImpactCandidates = append(d.utilityImpactCandidates, utilityImpactCandidate{id: 43, ownerMatches: true})
		}},
		{"nonfatal hurt", func(_ *Analyzer, d *Damage, _ *Kill) { d.VictimNewHealth = 99 }},
		{"zero HP removed", func(_ *Analyzer, d *Damage, _ *Kill) { d.HealthDamage = 0 }},
		{"duplicate fatal hurts", func(a *Analyzer, d *Damage, _ *Kill) { a.match.Damages = append(a.match.Damages, d) }},
		{"duplicate kills", func(a *Analyzer, _ *Damage, k *Kill) { a.match.Kills = append(a.match.Kills, k) }},
		{"different death tick", func(_ *Analyzer, _ *Damage, k *Kill) { k.Tick++ }},
		{"different death frame", func(_ *Analyzer, _ *Damage, k *Kill) { k.Frame++ }},
		{"different death victim", func(_ *Analyzer, _ *Damage, k *Kill) { k.VictimSteamID64++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			analyzer, damage, kill := utilityImpactLifecycleFixture()
			tc.change(analyzer, damage, kill)
			analyzer.resolveUtilityImpactLifecycles()
			if kill.HasUtilityImpactData || kill.IsUtilityImpactKill || kill.UtilityImpactSource != utilityImpactSourceUnavailable || kill.UtilityImpactProjectileID != "" {
				t.Fatalf("uncertain death was classified: %+v", kill)
			}
		})
	}
}

func TestUtilityImpactLifecycleIgnoresOnlyKnownExpiredGhosts(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		analyzer, damage, kill := utilityImpactLifecycleFixture()
		old := *analyzer.utilityImpactLifetimes[42]
		old.id, old.throwTick, old.throwFrame = 41, 50, 50
		old.effectTick, old.effectFrame = 120, 120
		analyzer.utilityImpactLifetimes[41] = &old
		ghost := utilityImpactCandidate{id: 41, ownerMatches: true}
		if reverse {
			damage.utilityImpactCandidates = append([]utilityImpactCandidate{ghost}, damage.utilityImpactCandidates...)
		} else {
			damage.utilityImpactCandidates = append(damage.utilityImpactCandidates, ghost)
		}
		analyzer.resolveUtilityImpactLifecycles()
		if !kill.IsUtilityImpactKill || kill.UtilityImpactProjectileID != "42" {
			t.Fatal("expired render entity confused a later throw of the same weapon")
		}
	}
}

func TestUtilityImpactLifecycleExports(t *testing.T) {
	analyzer, _, kill := utilityImpactLifecycleFixture()
	analyzer.match.Kills[0].setUtilityImpactData("ak47")
	analyzer.resolveUtilityImpactLifecycles()
	encoded, err := json.Marshal(kill)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["utilityImpactSource"] != utilityImpactSourceLifecycle || decoded["utilityImpactProjectileId"] != "42" ||
		decoded["utilityImpactEffectTick"] != float64(228) || decoded["utilityImpactEffect"] != "he_explode" {
		t.Fatal("JSON lifecycle provenance missing")
	}
	dir := t.TempDir()
	if err := exportMatchToCSV(analyzer.match, dir); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "utility-kills_kills.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][2]string{
		"utility impact source":        {utilityImpactSourceRawWeapon, utilityImpactSourceLifecycle},
		"utility impact projectile id": {"", "42"},
		"utility impact effect tick":   {"", "228"},
		"utility impact effect":        {"", "he_explode"},
	}
	for col, name := range rows[0] {
		if values, exists := want[name]; exists {
			if rows[1][col] != values[0] || rows[2][col] != values[1] {
				t.Fatalf("CSV lifecycle evidence: %s", name)
			}
			delete(want, name)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing CSV columns: %v", want)
	}
}

// Optional real-parser regression: Cache contains two pre-explosion HE impacts,
// although neither is fatal. Fatal propagation is covered above for HE/decoy.
func TestUtilityImpactLifecycleCacheDemo(t *testing.T) {
	path := os.Getenv("CSDA_UTILITY_IMPACT_CACHE_DEMO")
	if path == "" {
		t.Skip("set CSDA_UTILITY_IMPACT_CACHE_DEMO to the Aurora-MOUZ Cache demo")
	}
	match, err := AnalyzeDemo(path, AnalyzeDemoOptions{Source: constants.DemoSourceValve})
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]int{26987: 26988, 99682: 99690}
	for _, damage := range match.Damages {
		if evidence := damage.utilityImpactLifecycle; evidence != nil {
			expected, exists := want[damage.Tick]
			if !exists || evidence.effectTick != expected || damage.WeaponName != constants.WeaponHEGrenade || damage.VictimNewHealth == 0 {
				t.Fatalf("unexpected real impact: %+v / %+v", damage, evidence)
			}
			t.Logf("HE impact tick=%d projectile=%s explosion=%d", damage.Tick, evidence.projectileID, evidence.effectTick)
			delete(want, damage.Tick)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing real impacts: %v", want)
	}
	for _, player := range match.PlayersBySteamID {
		if player.UtilityImpactDeathCount() != 0 {
			t.Fatal("nonfatal HE impacts entered a death total")
		}
	}
}
