package api

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

func TestClutchAggregatesOnlyCountPlayerAndOpponentCount(t *testing.T) {
	match := &Match{Clutches: []*Clutch{
		{ClutcherSteamID64: 1, OpponentCount: 2, HasWon: true},
		{ClutcherSteamID64: 1, OpponentCount: 2, HasWon: false},
		{ClutcherSteamID64: 1, OpponentCount: 3, HasWon: false},
		{ClutcherSteamID64: 2, OpponentCount: 1, HasWon: true},
		{ClutcherSteamID64: 2, OpponentCount: 2, HasWon: true},
	}}
	player := &Player{SteamID64: 1, match: match}
	for opponents := 1; opponents <= 5; opponents++ {
		want := 0
		if opponents == 2 {
			want = 2
		}
		if opponents == 3 {
			want = 1
		}
		if got := len(player.oneVsXClutches(opponents)); got != want {
			t.Fatalf("1v%d count: got %d, want %d", opponents, got, want)
		}
		if got := player.oneVsXClutchesWonCount(opponents) + player.oneVsXClutchesLostCount(opponents); got != want {
			t.Fatalf("1v%d wins + losses: got %d, want %d", opponents, got, want)
		}
	}
	if player.OneVsTwoWonCount() != 1 || player.OneVsTwoLostCount() != 1 {
		t.Fatal("1v2 wins/losses included another player or duplicated events")
	}
}

func TestAwpHoldDeathsUseKillStateRegardlessOfPositionExport(t *testing.T) {
	for _, scoped := range []bool{true, false} {
		kill := &Kill{Frame: 100, Tick: 100, RoundNumber: 1,
			KillerSteamID64: 1, VictimSteamID64: 2, KillerX: 100,
			VictimActiveWeaponName: constants.WeaponAWP, VictimIsScoped: scoped,
		}
		match := &Match{TickRate: 64, FrameRate: 64, Kills: []*Kill{kill}}
		generateAwpHoldDeaths(match)
		before, _ := json.Marshal(match.AwpHoldDeaths)
		want := 0
		if scoped {
			want = 1
		}
		if len(match.AwpHoldDeaths) != want {
			t.Fatalf("scoped=%v: expected %d holds", scoped, want)
		}
		// Both an obsolete alive snapshot and a same-tick dead snapshot must not
		// override the event's weapon, scope, yaw, velocity, or coordinates.
		match.PlayerPositions = []*PlayerPosition{
			{RoundNumber: 1, Frame: 99, Tick: 99, SteamID64: 2, IsAlive: true, IsScoping: true, ActiveWeaponName: constants.WeaponAWP, X: 900, Yaw: 180},
			{RoundNumber: 1, Frame: 100, Tick: 100, SteamID64: 2, IsAlive: false, ActiveWeaponName: constants.WeaponUnknown},
		}
		generateAwpHoldDeaths(match)
		after, _ := json.Marshal(match.AwpHoldDeaths)
		if string(before) != string(after) {
			t.Fatalf("position export changed AWP holds: %s -> %s", before, after)
		}
	}
}

func TestWallbangUsesHurtPositionAndHonorsObservedZeroPenetrations(t *testing.T) {
	damage := &Damage{RoundNumber: 1, Frame: 100, Tick: 100, AttackerSteamID64: 1, VictimSteamID64: 2,
		WeaponName: "AK-47", WeaponUniqueID: "ak", HitGroup: events.HitGroupChest,
		HealthDamage: 12, VictimHealth: 100, victimPosition: &PlayerPosition{},
	}
	match := &Match{Damages: []*Damage{damage}, Shots: []*Shot{
		{RoundNumber: 1, Frame: 100, Tick: 100, PlayerSteamID64: 1, WeaponID: "ak"},
	}}
	markHeuristicWallbangDamages(match)
	if !damage.IsWallbang || damage.WallbangDataSource != "heuristic" {
		t.Fatal("damage-event position must enable heuristic without -positions")
	}
	match.PlayerPositions = []*PlayerPosition{{RoundNumber: 1, Frame: 100, Tick: 100, SteamID64: 2, X: 999999}}
	markHeuristicWallbangDamages(match)
	if !damage.IsWallbang {
		t.Fatal("frame-end position replaced hurt-event position")
	}
	damage.setBulletDamageData(0)
	markHeuristicWallbangDamages(match)
	if damage.IsWallbang || damage.WallbangDataSource != "bullet_damage" {
		t.Fatal("observed zero penetrations must override heuristic")
	}
	damage.HasBulletDamageData = false
	damage.victimPosition = nil
	match.PlayerPositions = nil
	markHeuristicWallbangDamages(match)
	if damage.IsWallbang || damage.WallbangDataSource != "unavailable" {
		t.Fatal("missing position must not imply zero-distance evidence")
	}
}

func TestDataQualityDistinguishesMissingFromObservedZero(t *testing.T) {
	match := &Match{Shots: []*Shot{{}, {HasAimPunchData: true, HasViewPunchData: true, HasRecoilIndexData: true}},
		Damages: []*Damage{{}, {HasBulletDamageData: true}},
	}
	quality := match.DataQuality()
	if quality.FootstepsStatus != "not_observed" || quality.AimPunchSampleCount != 1 || quality.BulletDamageMatchedCount != 1 {
		t.Fatalf("incorrect availability: %+v", quality)
	}
	bytes, err := json.Marshal(match)
	if err != nil {
		t.Fatal(err)
	}
	var exported map[string]any
	if err := json.Unmarshal(bytes, &exported); err != nil {
		t.Fatal(err)
	}
	shots := exported["shots"].([]any)
	if reflect.DeepEqual(shots[0], shots[1]) {
		t.Fatal("missing and observed zero serialized identically")
	}
	match.Footsteps = []*Footstep{{}}
	if match.DataQuality().FootstepsStatus != "observed" {
		t.Fatal("observed footsteps not reported")
	}
}

func TestWallbangHeadshotUsesObservedHelmetInsteadOfInferringFromArmor(t *testing.T) {
	// A player can have armor without a helmet. The old snapshot fallback inferred
	// a helmet from armor when the event sample had no recent frame timestamp.
	damage := &Damage{RoundNumber: 1, Frame: 100, Tick: 100, AttackerSteamID64: 1, VictimSteamID64: 2,
		WeaponName: "M4A1", WeaponUniqueID: "m4", HitGroup: events.HitGroupHead,
		HealthDamage: 96, VictimHealth: 100, VictimArmor: 100,
		victimPosition: &PlayerPosition{HasHelmet: false},
	}
	match := &Match{Damages: []*Damage{damage}, Shots: []*Shot{
		{RoundNumber: 1, Frame: 100, Tick: 100, PlayerSteamID64: 1, WeaponID: "m4"},
	}}
	markHeuristicWallbangDamages(match)
	if !damage.IsWallbang || damage.WallbangDataSource != "heuristic" {
		t.Fatal("armor alone must not imply a helmet")
	}
	damage.victimPosition.HasHelmet = true
	markHeuristicWallbangDamages(match)
	if damage.IsWallbang {
		t.Fatal("observed helmet must change expected headshot damage")
	}
}

func TestGameModeRequiresEvidence(t *testing.T) {
	match := &Match{}
	if match.GameModeStr() != "unknown" || match.GameModeSource() != "unknown" {
		t.Fatal("missing convars inferred casual")
	}
	match.HasGameTypeData = true
	if match.GameModeStr() != "unknown" {
		t.Fatal("partial convars inferred mode")
	}
	match.HasGameModeData = true
	if match.GameModeStr() != constants.GameModeStrCasual || match.GameModeSource() != "convars" {
		t.Fatal("observed zero convars must resolve casual")
	}
	match.gameModeStr = constants.GameModeStrCompetitive
	if match.GameModeSource() != "source_hint" {
		t.Fatal("source hint must be identified")
	}
	match.gameModeSource = "server_info"
	if match.GameModeSource() != "server_info" {
		t.Fatal("server info provenance missing")
	}
}
