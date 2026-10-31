package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	dem "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"google.golang.org/protobuf/proto"
)

func TestUtilityImpactPreservesCauseAndAmbiguity(t *testing.T) {
	for _, tc := range []struct {
		raw           string
		known, impact bool
	}{
		{"smokegrenade", true, true}, {"flashbang", true, true},
		{"molotov", true, true}, {"incgrenade", true, true},
		{"weapon_smokegrenade_projectile", true, true}, {"molotov_projectile", true, true},
		{"weapon_FLASHBANG", true, true},
		{"inferno", true, false}, {"weapon_inferno", true, false},
		{"ak47", true, false}, {"c4", true, false}, {"world", true, false},
		{"hegrenade", false, false}, {"hegrenade_projectile", false, false},
		{"decoy", false, false}, {"decoy_projectile", false, false},
		{"", false, false}, {"unrecognized", false, false}, {"custom_molotov", false, false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			// Fire damage may map to either fire grenade family. Raw event names,
			// not that mapped family or a small damage amount, decide the cause.
			kill := &Kill{WeaponName: constants.WeaponMolotov, HasUtilityImpactData: true, IsUtilityImpactKill: true}
			kill.setUtilityImpactData(tc.raw)
			if kill.RawWeaponName != tc.raw || kill.HasUtilityImpactData != tc.known || kill.IsUtilityImpactKill != tc.impact {
				t.Fatalf("incorrect impact evidence: %+v", kill)
			}
		})
	}
}

func TestUtilityImpactDeathCountRequiresEnemyAndKnownImpact(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Kill)
		want   int
	}{
		{"enemy", func(*Kill) {}, 1},
		{"teammate", func(k *Kill) { k.KillerSide = k.VictimSide }, 0},
		{"suicide", func(k *Kill) { k.KillerSteamID64 = k.VictimSteamID64 }, 0},
		{"world", func(k *Kill) { k.KillerSteamID64 = 0 }, 0},
		{"spectator", func(k *Kill) { k.KillerSide = common.TeamSpectators }, 0},
		{"other victim", func(k *Kill) { k.VictimSteamID64 = 4 }, 0},
		{"ambiguous HE", func(k *Kill) { k.setUtilityImpactData("hegrenade") }, 0},
		{"fire", func(k *Kill) { k.setUtilityImpactData("inferno") }, 0},
		{"unavailable evidence", func(k *Kill) { k.HasUtilityImpactData = false }, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			analyzer, kill := utilityKillFixture()
			kill.setUtilityImpactData("smokegrenade")
			tc.change(kill)
			analyzer.match.Kills = append(analyzer.match.Kills, nil, kill)
			if got := (&Player{SteamID64: 3, match: analyzer.match}).UtilityImpactDeathCount(); got != tc.want {
				t.Fatalf("count = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestUtilityImpactRawEventConsumesOnlyPendingSameFrameKill(t *testing.T) {
	parser := dem.NewParser(bytes.NewReader(nil))
	defer parser.Close()
	analyzer := &Analyzer{parser: parser}
	newKill := func() *Kill { return &Kill{Tick: analyzer.currentTick(), Frame: parser.CurrentFrame()} }
	event := events.GenericGameEvent{Name: "player_death", Data: map[string]*msg.CMsgSource1LegacyGameEventKeyT{
		"weapon": {ValString: proto.String("smokegrenade")},
	}}
	kill := newKill()
	analyzer.pendingUtilityImpactKill = kill
	analyzer.applyUtilityImpactDeathEvent(events.GenericGameEvent{Name: "player_hurt"})
	if analyzer.pendingUtilityImpactKill != kill {
		t.Fatal("unrelated event consumed pending death")
	}
	analyzer.applyUtilityImpactDeathEvent(event)
	if !kill.HasUtilityImpactData || !kill.IsUtilityImpactKill || analyzer.pendingUtilityImpactKill != nil {
		t.Fatal("raw death was not paired with its typed kill")
	}
	for _, change := range []func(*Kill){func(k *Kill) { k.Tick-- }, func(k *Kill) { k.Frame-- }} {
		kill = newKill()
		change(kill)
		analyzer.pendingUtilityImpactKill = kill
		analyzer.applyUtilityImpactDeathEvent(event)
		if kill.HasUtilityImpactData || analyzer.pendingUtilityImpactKill != nil {
			t.Fatal("stale pending kill was annotated or retained")
		}
	}
	// Missing raw data is unknown, and duplicate generic events have no target.
	kill = newKill()
	analyzer.pendingUtilityImpactKill = kill
	analyzer.applyUtilityImpactDeathEvent(events.GenericGameEvent{Name: "player_death"})
	analyzer.applyUtilityImpactDeathEvent(event)
	if kill.HasUtilityImpactData || kill.RawWeaponName != "" {
		t.Fatal("missing raw death data was inferred from another event")
	}
}

func TestUtilityImpactKillExports(t *testing.T) {
	analyzer, kill := utilityKillFixture()
	analyzer.match.Kills[0].setUtilityImpactData("ak47")
	kill.WeaponName = constants.WeaponSmoke
	kill.setUtilityImpactData("smokegrenade")
	analyzer.match.Kills = append(analyzer.match.Kills, kill)
	unknown := *kill
	unknown.Tick++
	unknown.WeaponName = constants.WeaponHEGrenade
	unknown.setUtilityImpactData("hegrenade")
	analyzer.match.Kills = append(analyzer.match.Kills, &unknown)
	encoded, err := json.Marshal(analyzer.match.Kills)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded[1]["rawWeaponName"] != "smokegrenade" || decoded[1]["hasUtilityImpactData"] != true ||
		decoded[1]["isUtilityImpactKill"] != true || decoded[2]["hasUtilityImpactData"] != false || decoded[2]["isUtilityImpactKill"] != false {
		t.Fatal("JSON impact evidence was lost")
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
	reader := csv.NewReader(f)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][3]string{
		"raw weapon name":         {"ak47", "smokegrenade", "hegrenade"},
		"has utility impact data": {"1", "1", "0"},
		"is utility impact kill":  {"0", "1", "0"},
	}
	for col, name := range rows[0] {
		if values, exists := want[name]; exists {
			for row, value := range values {
				if rows[row+1][col] != value {
					t.Fatalf("CSV %s row %d", name, row)
				}
			}
			delete(want, name)
		}
	}
	if len(want) > 0 {
		t.Fatalf("missing columns: %v", want)
	}
}
