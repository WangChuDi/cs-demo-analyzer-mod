package api

import (
	"encoding/binary"
	"hash/crc32"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

func testVBKV(payload []byte) []byte {
	data := append([]byte("VBKV\x00\x00\x00\x00"), payload...)
	binary.LittleEndian.PutUint32(data[4:8], crc32.ChecksumIEEE(payload))
	return data
}

func TestServerInfoActiveGameMode(t *testing.T) {
	payload := []byte("\x00\x00\x02c_game_type\x00\x00\x00\x00\x00\x02c_game_mode\x00\x01\x00\x00\x00\x02default_game_mode\x00\x00\x00\x00\x00\x0b\x0b")
	data := testVBKV(payload)
	match := &Match{gameModeStr: constants.GameModeStrCasual}
	match.applyServerInfo(&msg.CSVCMsg_ServerInfo{GameSessionConfig: &msg.CSVCMsg_GameSessionConfiguration{Data: data}})
	if !match.HasGameModeData || !match.HasGameTypeData || match.GameModeStr() != constants.GameModeStrCompetitive || match.GameModeSource() != "server_info_data" {
		t.Fatalf("active session mode not recovered: %+v / %s / %s", match, match.GameModeStr(), match.GameModeSource())
	}
	for _, bad := range [][]byte{
		data[:len(data)-1],                                          // bad checksum and truncated document
		testVBKV(payload[:len(payload)-1]),                          // valid checksum, truncated document
		testVBKV(append(append([]byte{}, payload...), 1)),           // trailing garbage
		testVBKV([]byte("\x02c_game_type\x00\x00\x00\x00\x00\x0b")), // partial pair
		testVBKV([]byte("\x02c_game_type\x00\x00\x00\x00\x00\x02c_game_mode\x00\x01\x00\x00\x00\x02c_game_mode\x00\x00\x00\x00\x00\x0b")), // duplicate
		testVBKV([]byte("\x00nested\x00\x02c_game_type\x00\x00\x00\x00\x00\x02c_game_mode\x00\x01\x00\x00\x00\x0b\x0b")),                  // irrelevant subtree
	} {
		if _, _, ok := gameModeFromVBKV(bad); ok {
			t.Fatal("accepted malformed or ambiguous session data")
		}
	}
}

func TestFootstepSoundNamesAgainstRecordedHashes(t *testing.T) {
	// Independent observed pairs from the 2026-10-29 Cache recording.
	for hash, name := range map[uint32]string{1194677450: "CT_Concrete.StepLeft", 4163677892: "T_Concrete.StepLeft", 3166948458: "CT_Gravel.StepLeft", 1218015996: "T_Gravel.StepLeft", 3749333696: "CT_Grass.StepLeft", 3023174225: "T_WetConcrete.StepLeft"} {
		if got := footstepSoundNames[hash]; got != name {
			t.Fatalf("hash %d: got %s, want %s", hash, got, name)
		}
	}
	for _, name := range []string{"Land_Concrete.StepLeft", "Gear.JumpLand.CT", "Weapon_AK47.Single", "Unknown.StepLeft"} {
		if _, ok := footstepSoundNames[soundNameHash(name)]; ok {
			t.Fatalf("classified non-footstep %s", name)
		}
	}
	if soundNameHash("CT_Concrete.StepLeft") != soundNameHash("ct_concrete.stepleft") {
		t.Fatal("sound names must be normalized")
	}
}

func TestKillPenetrationMatchesOnlyUnambiguousLethalDamage(t *testing.T) {
	damage := func(frame int, health int) *Damage {
		return &Damage{Frame: frame, RoundNumber: 1, AttackerSteamID64: 10, VictimSteamID64: 20, HealthDamage: 12, VictimNewHealth: health, WeaponName: constants.WeaponAK47}
	}
	kill := func(frame int) *Kill {
		return &Kill{Frame: frame, RoundNumber: 1, KillerSteamID64: 10, VictimSteamID64: 20, PenetratedObjects: 2, WeaponName: constants.WeaponAK47}
	}
	prior, lethal, otherFrame := damage(1, 10), damage(2, 0), damage(3, 0)
	ambiguous1, ambiguous2 := damage(4, 0), damage(4, 0)
	observed := damage(5, 0)
	observed.setBulletDamageData(0)
	wrongWeapon := damage(6, 0)
	wrongWeapon.WeaponName = constants.WeaponAWP
	match := &Match{Damages: []*Damage{prior, lethal, otherFrame, ambiguous1, ambiguous2, observed, wrongWeapon}, Kills: []*Kill{kill(2), kill(4), kill(5), kill(6)}}
	matchKillPenetrations(match)
	markHeuristicWallbangDamages(match)
	if !lethal.HasKillPenetrationData || lethal.HasBulletDamageData || lethal.NumPenetrations != 2 || !lethal.IsWallbang || lethal.WallbangDataSource != "kill_event" {
		t.Fatal("fatal penetration not recovered with correct provenance")
	}
	for _, d := range []*Damage{prior, otherFrame, ambiguous1, ambiguous2, observed, wrongWeapon} {
		if d.HasKillPenetrationData {
			t.Fatal("matched ambiguous, nonlethal, unrelated, or authoritative damage")
		}
	}
	if observed.IsWallbang || observed.WallbangDataSource != "bullet_damage" {
		t.Fatal("overrode observed zero")
	}
}
