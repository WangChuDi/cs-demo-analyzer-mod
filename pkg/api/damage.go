package api

import (
	"fmt"

	"github.com/akiver/cs-demo-analyzer/internal/math"
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

type Damage struct {
	// Frame is retained internally for fork event correlation, not exported.
	Frame        int `json:"-"`
	Tick         int `json:"tick"`
	RoundNumber  int `json:"roundNumber"`
	HealthDamage int `json:"healthDamage"`
	ArmorDamage  int `json:"armorDamage"`
	// IsWallbang is the outward/public wallbang flag.
	//
	// It is true when either:
	//  1. direct penetration is available (BulletDamage, PlayerBulletHit or an unambiguous fatal Kill), or
	//  2. heuristic fallback classifies the damage as wallbang (wallbang.go), used when direct non-lethal extraction is unavailable.
	IsWallbang bool `json:"isWallbang"`
	// WallbangDataSource is bullet_damage, player_bullet_hit, kill_event, heuristic, or unavailable. A false
	// IsWallbang with source unavailable is not evidence of an unobstructed shot.
	WallbangDataSource       string               `json:"wallbangDataSource"`
	HasBulletDamageData      bool                 `json:"hasBulletDamageData"`
	HasKillPenetrationData   bool                 `json:"hasKillPenetrationData"`
	HasPlayerBulletHitData   bool                 `json:"hasPlayerBulletHitData"`
	NumPenetrations          int                  `json:"numPenetrations"`
	AttackerSteamID64        uint64               `json:"attackerSteamId"`
	AttackerSide             common.Team          `json:"attackerSide"`
	AttackerTeamName         string               `json:"attackerTeamName"`
	IsAttackerControllingBot bool                 `json:"isAttackerControllingBot"`
	VictimHealth             int                  `json:"victimHealth"`
	VictimNewHealth          int                  `json:"victimNewHealth"`
	VictimArmor              int                  `json:"victimArmor"`
	VictimNewArmor           int                  `json:"victimNewArmor"`
	VictimSteamID64          uint64               `json:"victimSteamId"`
	VictimSide               common.Team          `json:"victimSide"`
	VictimTeamName           string               `json:"victimTeamName"`
	IsVictimControllingBot   bool                 `json:"isVictimControllingBot"`
	HitGroup                 events.HitGroup      `json:"hitgroup"`
	WeaponName               constants.WeaponName `json:"weaponName"`
	WeaponType               constants.WeaponType `json:"weaponType"`
	WeaponUniqueID           string               `json:"weaponUniqueId"`
	IsVictimAirborne         bool                 `json:"isVictimAirborne"`
	IsAttackerAirborne       bool                 `json:"isAttackerAirborne"`
	victimPosition           *PlayerPosition
	isFallDamage             bool
}

type damageMatchFrameKey struct {
	roundNumber       int
	frame             int
	attackerSteamID64 uint64
	victimSteamID64   uint64
}

func (damage *Damage) IsGrenadeWeapon() bool {
	return damage.WeaponType == constants.WeaponTypeGrenade
}

func (damage *Damage) isValidPlayerDamageEvent(player *Player) bool {
	if damage.AttackerSteamID64 != player.SteamID64 {
		return false
	}
	if damage.AttackerSteamID64 == damage.VictimSteamID64 {
		return false
	}
	if damage.VictimSteamID64 == 0 {
		return false
	}
	if damage.IsAttackerControllingBot {
		return false
	}
	if damage.AttackerSide == damage.VictimSide {
		return false
	}

	return true
}

func (damage *Damage) isWallbang() bool {
	return (damage.HasBulletDamageData || damage.HasPlayerBulletHitData || damage.HasKillPenetrationData) && damage.NumPenetrations > 0
}

func (damage *Damage) setBulletDamageData(numPenetrations int) {
	damage.HasBulletDamageData = true
	damage.NumPenetrations = numPenetrations
}

func newDamageMatchFrameKey(roundNumber int, frame int, attackerSteamID64 uint64, victimSteamID64 uint64) damageMatchFrameKey {
	return damageMatchFrameKey{
		roundNumber:       roundNumber,
		frame:             frame,
		attackerSteamID64: attackerSteamID64,
		victimSteamID64:   victimSteamID64,
	}
}

func newDamageMatchFrameKeyFromDamage(damage *Damage) damageMatchFrameKey {
	return newDamageMatchFrameKey(damage.RoundNumber, damage.Frame, damage.AttackerSteamID64, damage.VictimSteamID64)
}

func newDamageFromGameEvent(analyzer *Analyzer, event events.PlayerHurt) *Damage {
	if event.Weapon == nil {
		fmt.Println("Player hurt event without weapon occurred")
		return nil
	}
	match := analyzer.match
	attackerSteamID := uint64(0)
	attackerSide := common.TeamUnassigned
	attackerTeamName := "World"
	isAttackerControllingBot := false
	var isAttackerAirborne bool
	if event.Attacker != nil {
		attackerSteamID = event.Attacker.SteamID64
		attackerSide = event.Attacker.Team
		attackerTeamName = match.Team(event.Attacker.Team).Name
		isAttackerControllingBot = event.Attacker.IsControllingBot()
		isAttackerAirborne = event.Attacker.IsAirborne()
	}
	// Only retain the state needed by the damage model, independently of the
	// optional per-frame position export (and before frame-end/death updates).
	var victimPosition *PlayerPosition
	if (!analyzer.isSource2 && event.Player.Entity != nil) || (analyzer.isSource2 && event.Player.PlayerPawnEntity() != nil) {
		position := event.Player.Position()
		victimPosition = &PlayerPosition{X: position.X, Y: position.Y, Z: position.Z, HasHelmet: event.Player.HasHelmet()}
	}

	return &Damage{
		Frame:                    analyzer.parser.CurrentFrame(),
		RoundNumber:              analyzer.currentRound.Number,
		Tick:                     analyzer.currentTick(),
		HealthDamage:             math.Max(0, event.HealthDamageTaken),
		ArmorDamage:              math.Max(0, event.ArmorDamageTaken),
		VictimHealth:             math.Max(0, event.Player.Health()),
		VictimArmor:              math.Max(0, event.Player.Armor()),
		VictimNewHealth:          math.Max(0, event.Health),
		VictimNewArmor:           math.Max(0, event.Armor),
		IsVictimControllingBot:   event.Player.IsControllingBot(),
		AttackerSteamID64:        attackerSteamID,
		AttackerSide:             attackerSide,
		AttackerTeamName:         attackerTeamName,
		IsAttackerControllingBot: isAttackerControllingBot,
		VictimSteamID64:          event.Player.SteamID64,
		VictimSide:               event.Player.Team,
		VictimTeamName:           match.Team(event.Player.Team).Name,
		WeaponName:               equipmentToWeaponName[event.Weapon.Type],
		WeaponType:               getEquipmentWeaponType(*event.Weapon),
		HitGroup:                 event.HitGroup,
		WeaponUniqueID:           event.Weapon.UniqueID().String(),
		IsVictimAirborne:         event.Player.IsAirborne(),
		IsAttackerAirborne:       isAttackerAirborne,
		victimPosition:           victimPosition,
	}
}

func newFallDamageFromGenericPlayerHurt(analyzer *Analyzer, event events.GenericGameEvent) *Damage {
	if event.Name != "player_hurt" {
		return nil
	}

	userIDData, exists := event.Data["userid"]
	if !exists || userIDData == nil {
		return nil
	}

	playerUserID := int(userIDData.GetValShort())
	victim := analyzer.parser.GameState().Participants().ByUserID()[playerUserID]
	if victim == nil {
		return nil
	}

	healthDamage := math.Max(0, genericGameEventInt(event, "dmg_health"))
	armorDamage := math.Max(0, genericGameEventInt(event, "dmg_armor"))

	match := analyzer.match
	newHealth := math.Max(0, genericGameEventInt(event, "health"))
	newArmor := math.Max(0, genericGameEventInt(event, "armor"))

	return &Damage{
		RoundNumber:              analyzer.currentRound.Number,
		Frame:                    analyzer.parser.CurrentFrame(),
		Tick:                     analyzer.currentTick(),
		HealthDamage:             healthDamage,
		ArmorDamage:              armorDamage,
		VictimHealth:             math.Max(0, newHealth+healthDamage),
		VictimArmor:              math.Max(0, newArmor+armorDamage),
		VictimNewHealth:          newHealth,
		VictimNewArmor:           newArmor,
		IsVictimControllingBot:   victim.IsControllingBot(),
		AttackerSteamID64:        0,
		AttackerSide:             common.TeamUnassigned,
		AttackerTeamName:         "World",
		IsAttackerControllingBot: false,
		VictimSteamID64:          victim.SteamID64,
		VictimSide:               victim.Team,
		VictimTeamName:           match.Team(victim.Team).Name,
		HitGroup:                 events.HitGroupGeneric,
		WeaponName:               constants.WeaponWorld,
		WeaponType:               constants.WeaponTypeWorld,
		WeaponUniqueID:           "",
		IsVictimAirborne:         victim.IsAirborne(),
		IsAttackerAirborne:       false,
		isFallDamage:             true,
	}
}

func genericGameEventInt(event events.GenericGameEvent, key string) int {
	v, exists := event.Data[key]
	if !exists || v == nil {
		return 0
	}

	return int(v.GetValShort())
}
