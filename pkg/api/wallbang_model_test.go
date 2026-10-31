package api

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

func wallbangModelFixture() (*Match, *Damage) {
	d := &Damage{Frame: 100, Tick: 100, RoundNumber: 1, AttackerSteamID64: 1, VictimSteamID64: 2,
		AttackerSide: common.TeamTerrorists, VictimSide: common.TeamCounterTerrorists,
		WeaponName: constants.WeaponAK47, WeaponUniqueID: "ak", HitGroup: events.HitGroupChest,
		VictimHealth: 100, VictimNewHealth: 65, HealthDamage: 35,
		ReportedHealthDamage: 35, HasReportedHealthDamage: true, victimPosition: &PlayerPosition{},
	}
	return &Match{NetworkProtocol: 14191, Damages: []*Damage{d}, Shots: []*Shot{
		{Frame: 100, Tick: 100, RoundNumber: 1, PlayerSteamID64: 1, WeaponID: "ak"},
	}}, d
}

func TestWallbangIntegerReportComparison(t *testing.T) {
	for _, tc := range []struct {
		name                                       string
		distance                                   float64
		armor, reported, actual, expected, deficit int
		wallbang                                   bool
		reason                                     string
	}{
		{"one point loss is ignored", 0, 0, 35, 35, 36, 1, false, "within_one_point_tolerance"},
		{"two point loss is a candidate", 0, 0, 34, 35, 36, 2, true, "damage_deficit"},
		{"fractional report is floored", 500, 0, 35, 35, 35, 0, false, "consistent_damage"},
		{"HP delta can exceed report", 500, 0, 35, 36, 35, 0, false, "consistent_damage"},
		{"one reported point lost even if HP delta matches baseline", 500, 0, 34, 35, 35, 1, false, "within_one_point_tolerance"},
		{"low armor exhausts", 0, 1, 34, 34, 34, 0, false, "consistent_damage"},
		{"low armor one point loss", 0, 1, 33, 33, 34, 1, false, "within_one_point_tolerance"},
		{"full armor", 0, 100, 27, 28, 27, 0, false, "consistent_damage"},
		{"higher observed is a model mismatch", 500, 0, 36, 36, 35, -1, false, "observed_above_expected"},
		{"observed zero is not missing", 0, 0, 0, 1, 36, 36, true, "damage_deficit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, d := wallbangModelFixture()
			d.victimPosition.X, d.VictimArmor = tc.distance, tc.armor
			d.ReportedHealthDamage, d.HealthDamage = tc.reported, tc.actual
			markHeuristicWallbangDamages(m)
			if d.IsWallbang != tc.wallbang || d.WallbangExpectedDamage != tc.expected || d.WallbangDamageDeficit != tc.deficit || d.WallbangHeuristicReason != tc.reason || !d.HasWallbangHeuristicData {
				t.Fatalf("incorrect comparison: %+v", d)
			}
		})
	}
}

func TestWallbangRequiresUsableInputs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Match, *Damage)
		reason string
	}{
		{"missing report", func(m *Match, d *Damage) { d.HasReportedHealthDamage = false }, "missing_reported_damage"},
		{"friendly fire", func(m *Match, d *Damage) { d.VictimSide = d.AttackerSide }, "friendly_fire"},
		{"self damage", func(m *Match, d *Damage) { d.VictimSteamID64 = d.AttackerSteamID64 }, "invalid_players"},
		{"generic hitgroup", func(m *Match, d *Damage) { d.HitGroup = events.HitGroupGeneric }, "unsupported_hitgroup"},
		{"shotgun pellets", func(m *Match, d *Damage) { d.WeaponName = constants.WeaponNova }, "unsupported_weapon_or_version"},
		{"unknown protocol", func(m *Match, d *Damage) { m.NetworkProtocol = 0 }, "unsupported_weapon_or_version"},
		{"old protocol", func(m *Match, d *Damage) { m.NetworkProtocol = 13990 }, "unsupported_weapon_or_version"},
		{"future protocol", func(m *Match, d *Damage) { m.NetworkProtocol = 14193 }, "unsupported_weapon_or_version"},
		{"previous frame shot", func(m *Match, d *Damage) { m.Shots[0].Frame-- }, "missing_or_ambiguous_shot"},
		{"multiple shots", func(m *Match, d *Damage) { s := *m.Shots[0]; m.Shots = append(m.Shots, &s) }, "missing_or_ambiguous_shot"},
		{"different weapon", func(m *Match, d *Damage) { m.Shots[0].WeaponID = "other" }, "missing_or_ambiguous_shot"},
		{"missing position", func(m *Match, d *Damage) { d.victimPosition = nil }, "missing_position"},
		{"nonfinite position", func(m *Match, d *Damage) { d.victimPosition.X = math.NaN() }, "invalid_position"},
		{"invalid convar", func(m *Match, d *Damage) {
			d.wallbangDamageScale = snapshotWallbangDamageScale(map[string]string{"mp_damage_scale_ct_body": "bad"}, d.VictimSide, d.HitGroup)
		}, "invalid_damage_scale"},
		{"zero convar", func(m *Match, d *Damage) {
			d.wallbangDamageScale = snapshotWallbangDamageScale(map[string]string{"mp_damage_scale_ct_body": "0"}, d.VictimSide, d.HitGroup)
		}, "invalid_damage_scale"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, d := wallbangModelFixture()
			tc.change(m, d)
			markHeuristicWallbangDamages(m)
			if d.IsWallbang || d.WallbangDataSource != "unavailable" || d.WallbangHeuristicReason != tc.reason {
				t.Fatalf("unsafe inference: %+v", d)
			}
		})
	}
}

func TestWallbangArmorAndDamageScales(t *testing.T) {
	m, d := wallbangModelFixture()
	d.VictimArmor = 1
	d.ReportedHealthDamage = 16 // 36*0.5 - 1/0.5; scale is applied BEFORE armor.
	d.wallbangDamageScale = snapshotWallbangDamageScale(map[string]string{"mp_damage_scale_ct_body": "0.5", "mp_damage_scale_t_body": "3", "mp_damage_scale_ct_head": "4"}, d.VictimSide, d.HitGroup)
	markHeuristicWallbangDamages(m)
	if d.IsWallbang || d.WallbangExpectedDamage != 16 {
		t.Fatalf("incorrect scale/armor order: %+v", d)
	}
	model := constants.HeuristicWallbangWeaponDamageModels[constants.WeaponAK47]
	for _, tc := range []struct {
		group  events.HitGroup
		armor  int
		helmet bool
		want   float64
	}{
		{events.HitGroupHead, 100, false, 144}, {events.HitGroupHead, 100, true, 111.6},
		{events.HitGroupStomach, 0, false, 45}, {events.HitGroupLeftLeg, 100, true, 27},
		{events.HitGroupChest, 1, false, 34}, {events.HitGroupChest, 4, false, 28}, {events.HitGroupChest, 5, false, 27.9},
	} {
		if got := expectedHealthDamageHeuristic(model, tc.group, 0, tc.armor, tc.helmet); math.Abs(got-tc.want) > 1e-8 {
			t.Fatalf("group %d armor %d: %.8f != %.8f", tc.group, tc.armor, got, tc.want)
		}
	}
}

func TestArmorBeforeHurtTracksSameFrameSequence(t *testing.T) {
	damages := []*Damage{{Frame: 99, VictimSteamID64: 2, VictimNewArmor: 90}, {Frame: 100, VictimSteamID64: 2, VictimNewArmor: 2}, {Frame: 100, VictimSteamID64: 3, VictimNewArmor: 50}}
	if got := armorBeforeHurt(damages, 100, 2, 5); got != 2 {
		t.Fatalf("second pellet used stale armor: %d", got)
	}
	damages = append(damages, &Damage{Frame: 100, VictimSteamID64: 2, VictimNewArmor: 0})
	if got := armorBeforeHurt(damages, 100, 2, 5); got != 0 {
		t.Fatalf("armor restored after depletion: %d", got)
	}
	if got := armorBeforeHurt(damages, 101, 2, 80); got != 80 {
		t.Fatalf("previous frame armor leaked: %d", got)
	}
}

func TestWallbangWeaponProfiles(t *testing.T) {
	for _, tc := range []struct {
		weapon           constants.WeaponName
		protocol         int
		damage, rangeMod float64
	}{
		{constants.WeaponMP7, 14129, 29, .85}, {constants.WeaponMP7, 14130, 30, .87},
		{constants.WeaponMP5, 14129, 27, .85}, {constants.WeaponMP5, 14130, 28, .87},
		{constants.WeaponMP9, 14191, 26, .87}, {constants.WeaponP2000, 14191, 35, .91},
	} {
		m, source, ok := constants.HeuristicWallbangModel(tc.weapon, tc.protocol)
		if !ok || source == "" || m.BaseDamage != tc.damage || m.RangeModifier != tc.rangeMod {
			t.Fatalf("incorrect profile for %s/%d: %+v", tc.weapon, tc.protocol, m)
		}
	}
	if len(constants.HeuristicWallbangWeaponDamageModels) != 30 {
		t.Fatal("single-projectile gun coverage changed")
	}
}

func TestWallbangDiagnosticsExport(t *testing.T) {
	m, d := wallbangModelFixture()
	ct, tt := common.TeamCounterTerrorists, common.TeamTerrorists
	m.DemoFileName = "wallbang-model"
	m.TeamA = &Team{CurrentSide: &ct}
	m.TeamB = &Team{CurrentSide: &tt}
	d.HealthDamage = 36 // A one-point report deficit is retained only as a diagnostic.
	markHeuristicWallbangDamages(m)
	p := &Player{SteamID64: 1, match: m}
	victim := &Player{SteamID64: 2, match: m}
	if p.WallbangDamageDealt() != 0 || victim.WallbangDamageTaken() != 0 || m.DataQuality().WallbangHeuristicEvaluatedCount != 1 {
		t.Fatal("one-point tolerance changed aggregates or hid the evaluation")
	}
	bytes, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err = json.Unmarshal(bytes, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["wallbangDamageDeficit"] != float64(1) || decoded["reportedHealthDamage"] != float64(35) || decoded["hasReportedHealthDamage"] != true || decoded["isWallbang"] != false || decoded["wallbangHeuristicReason"] != "within_one_point_tolerance" {
		t.Fatalf("missing JSON diagnostics: %s", bytes)
	}
	dir := t.TempDir()
	if err := exportMatchToCSV(m, dir); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "wallbang-model_damages.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || len(rows[0]) != len(rows[1]) {
		t.Fatal("misaligned damage export")
	}
	want := map[string]string{"health": "36", "reported health damage": "35", "has reported health damage": "1", "has wallbang heuristic data": "1", "wallbang expected damage": "36", "wallbang damage deficit": "1", "wallbang heuristic reason": "within_one_point_tolerance", "wallbang model source": "cs2_weapons_14130", "is wallbang": "0"}
	for i, name := range rows[0] {
		if expected, ok := want[name]; ok {
			if rows[1][i] != expected {
				t.Fatalf("%s = %q, want %q", name, rows[1][i], expected)
			}
			delete(want, name)
		}
	}
	if len(want) > 0 {
		t.Fatalf("missing CSV columns: %v", want)
	}
	d.ReportedHealthDamage = 34
	markHeuristicWallbangDamages(m)
	if !d.IsWallbang || d.WallbangDamageDeficit != 2 || p.WallbangDamageDealt() != 36 || victim.WallbangDamageTaken() != 36 {
		t.Fatal("two-point candidate did not aggregate actual HP damage")
	}
	d.ReportedHealthDamage = 35
	d.setBulletDamageData(1)
	markHeuristicWallbangDamages(m)
	if !d.IsWallbang || d.WallbangDataSource != "bullet_damage" || p.WallbangDamageDealt() != 36 {
		t.Fatal("one-point tolerance overrode confirmed penetration")
	}
	d.setBulletDamageData(0)
	markHeuristicWallbangDamages(m)
	if d.IsWallbang || d.HasWallbangHeuristicData || d.WallbangDamageDeficit != 0 || d.WallbangModelSource != "" {
		t.Fatal("direct zero did not clear inference")
	}
}

func TestWallbangEvaluationCountIncludesModelMismatch(t *testing.T) {
	m, d := wallbangModelFixture()
	d.ReportedHealthDamage = 37
	markHeuristicWallbangDamages(m)
	if d.WallbangDataSource != "unavailable" || !d.HasWallbangHeuristicData || m.DataQuality().WallbangHeuristicEvaluatedCount != 1 {
		t.Fatal("computed model mismatch was hidden from evaluation counts")
	}
}
