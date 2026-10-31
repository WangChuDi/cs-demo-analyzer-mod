package api

import (
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

const (
	utilityImpactSourceUnavailable = "unavailable"
	utilityImpactSourceRawWeapon   = "raw_weapon"
	utilityImpactSourceLifecycle   = "projectile_lifecycle"
)

type utilityImpactLifetime struct {
	id                        common.UniqueID
	entityID                  int
	weaponID                  string
	kind                      common.EquipmentType
	round                     int
	throwTick, throwFrame     int
	destroyTick, destroyFrame int
	destroyOwner              uint64
	effectTick, effectFrame   int
	effect                    string
	hasThrow, hasDestroy      bool
	hasEffect, ambiguous      bool
}

type utilityImpactCandidate struct {
	id           common.UniqueID
	ownerMatches bool
}

type utilityImpactLifecycleEvidence struct {
	projectileID string
	effectTick   int
	effect       string
}

func utilityImpactKind(weapon constants.WeaponName) common.EquipmentType {
	switch weapon {
	case constants.WeaponHEGrenade:
		return common.EqHE
	case constants.WeaponDecoy:
		return common.EqDecoy
	}
	return common.EqUnknown
}

func utilityImpactThrower(projectile *common.GrenadeProjectile) uint64 {
	for _, player := range []*common.Player{projectile.Thrower, projectile.Owner, projectile.WeaponInstance.Owner} {
		if player != nil && player.SteamID64 != 0 {
			return player.SteamID64
		}
	}
	return 0
}

func (analyzer *Analyzer) utilityImpactLifetime(projectile *common.GrenadeProjectile) *utilityImpactLifetime {
	if projectile == nil || projectile.Entity == nil || projectile.WeaponInstance == nil || analyzer.currentRound == nil {
		return nil
	}
	kind := projectile.WeaponInstance.Type
	if kind != common.EqHE && kind != common.EqDecoy {
		return nil
	}
	if analyzer.utilityImpactLifetimes == nil {
		analyzer.utilityImpactLifetimes = make(map[common.UniqueID]*utilityImpactLifetime)
	}
	id := projectile.UniqueID()
	if id == 0 {
		return nil
	}
	life := analyzer.utilityImpactLifetimes[id]
	if life == nil {
		life = &utilityImpactLifetime{id: id, entityID: projectile.Entity.ID(),
			weaponID: projectile.WeaponInstance.UniqueID().String(), kind: kind, round: analyzer.currentRound.Number}
		analyzer.utilityImpactLifetimes[id] = life
	}
	return life
}

func (analyzer *Analyzer) registerUtilityImpactLifecycleHandlers() {
	parser := analyzer.parser
	parser.RegisterEventHandler(func(event events.GrenadeProjectileThrow) {
		if !analyzer.matchStarted() {
			return
		}
		if life := analyzer.utilityImpactLifetime(event.Projectile); life != nil {
			if life.hasThrow {
				life.ambiguous = true
			}
			life.hasThrow = true
			life.throwTick, life.throwFrame = analyzer.currentTick(), parser.CurrentFrame()
		}
	})
	parser.RegisterEventHandler(func(event events.GrenadeProjectileDestroy) {
		if !analyzer.matchStarted() {
			return
		}
		if life := analyzer.utilityImpactLifetime(event.Projectile); life != nil {
			life.hasDestroy = true
			life.destroyTick, life.destroyFrame = analyzer.currentTick(), parser.CurrentFrame()
			life.destroyOwner = utilityImpactThrower(event.Projectile)
		}
	})
	parser.RegisterEventHandler(func(event events.HeExplode) {
		analyzer.recordUtilityImpactEffect(event.GrenadeEvent, common.EqHE, "he_explode")
	})
	parser.RegisterEventHandler(func(event events.DecoyExpired) {
		analyzer.recordUtilityImpactEffect(event.GrenadeEvent, common.EqDecoy, "decoy_expired")
	})
}

func (analyzer *Analyzer) recordUtilityImpactEffect(event events.GrenadeEvent, kind common.EquipmentType, effect string) {
	if !analyzer.matchStarted() || event.Projectile == nil || event.Grenade == nil {
		return
	}
	life := analyzer.utilityImpactLifetime(event.Projectile)
	if life == nil || life.kind != kind || life.entityID != event.GrenadeEntityID ||
		life.weaponID != event.Grenade.UniqueID().String() {
		return
	}
	tick, frame := analyzer.currentTick(), analyzer.parser.CurrentFrame()
	if life.hasEffect && (life.effectTick != tick || life.effectFrame != frame) {
		life.ambiguous = true
	}
	life.hasEffect, life.effectTick, life.effectFrame, life.effect = true, tick, frame, effect
}

// Snapshot every possible same-owner/type projectile, not just the parser's
// last-thrown equipment. Include same-tick/frame deletions even when the object
// has already left GameState. Unknown ownership remains a competing candidate.
func (analyzer *Analyzer) captureUtilityImpactCandidates(damage *Damage) {
	kind := utilityImpactKind(damage.WeaponName)
	if kind == common.EqUnknown || damage.AttackerSteamID64 == 0 || damage.IsAttackerControllingBot || damage.IsVictimControllingBot {
		return
	}
	seen := map[common.UniqueID]bool{}
	add := func(id common.UniqueID, owner uint64) {
		if seen[id] || (owner != 0 && owner != damage.AttackerSteamID64) {
			return
		}
		seen[id] = true
		damage.utilityImpactCandidates = append(damage.utilityImpactCandidates,
			utilityImpactCandidate{id: id, ownerMatches: owner == damage.AttackerSteamID64})
	}
	for _, projectile := range analyzer.parser.GameState().GrenadeProjectiles() {
		if projectile.WeaponInstance == nil || projectile.WeaponInstance.Type != kind {
			continue
		}
		add(projectile.UniqueID(), utilityImpactThrower(projectile))
	}
	for id, life := range analyzer.utilityImpactLifetimes {
		if life.kind == kind && life.round == damage.RoundNumber && life.hasDestroy &&
			(life.destroyFrame == damage.Frame || life.destroyTick == damage.Tick) {
			add(id, life.destroyOwner)
		}
	}
}

func (analyzer *Analyzer) resolveUtilityImpactDamage(damage *Damage) *utilityImpactLifecycleEvidence {
	if damage.AttackerSteamID64 == 0 || damage.VictimSteamID64 == 0 || damage.IsAttackerControllingBot || damage.IsVictimControllingBot {
		return nil
	}
	var selected *utilityImpactLifetime
	for _, candidate := range damage.utilityImpactCandidates {
		life := analyzer.utilityImpactLifetimes[candidate.id]
		if life == nil || life.id == 0 || life.id != candidate.id || !life.hasThrow || life.ambiguous || life.round != damage.RoundNumber ||
			life.kind != utilityImpactKind(damage.WeaponName) || life.throwTick > damage.Tick || life.throwFrame > damage.Frame {
			return nil
		}
		if life.hasEffect && (life.effect == "" || life.effectTick < life.throwTick || life.effectFrame < life.throwFrame ||
			(life.hasDestroy && (life.destroyTick < life.effectTick || life.destroyFrame < life.effectFrame))) {
			return nil
		}
		// Exploded entities may remain for five seconds. A known earlier effect
		// cannot be the source of a new pre-detonation impact.
		if life.hasEffect && life.effectTick < damage.Tick && life.effectFrame < damage.Frame {
			continue
		}
		if selected != nil || !candidate.ownerMatches {
			return nil
		}
		selected = life
	}
	if selected == nil || !selected.hasEffect || selected.effectTick <= damage.Tick || selected.effectFrame <= damage.Frame ||
		selected.weaponID == "" || selected.weaponID != damage.WeaponUniqueID {
		return nil
	}
	return &utilityImpactLifecycleEvidence{projectileID: selected.id.String(), effectTick: selected.effectTick, effect: selected.effect}
}

// Resolve after ParseToEnd: both the hurt and the later effect must have been
// observed. All comparisons use IngameTick/CurrentFrame, never raw server ticks
// from m_nExplodeEffectTickBegin or the time an entity eventually disappears.
func (analyzer *Analyzer) resolveUtilityImpactLifecycles() {
	type key struct {
		damageMatchFrameKey
		tick   int
		weapon constants.WeaponName
	}
	fatalDamages := map[key][]*Damage{}
	for _, damage := range analyzer.match.Damages {
		if damage == nil || utilityImpactKind(damage.WeaponName) == common.EqUnknown {
			continue
		}
		damage.utilityImpactLifecycle = analyzer.resolveUtilityImpactDamage(damage)
		if damage.VictimNewHealth == 0 && damage.HealthDamage > 0 {
			k := key{newDamageMatchFrameKeyFromDamage(damage), damage.Tick, damage.WeaponName}
			fatalDamages[k] = append(fatalDamages[k], damage)
		}
	}
	kills := map[key][]*Kill{}
	for _, kill := range analyzer.match.Kills {
		if kill != nil && utilityImpactKind(kill.WeaponName) != common.EqUnknown {
			k := key{newDamageMatchFrameKey(kill.RoundNumber, kill.Frame, kill.KillerSteamID64, kill.VictimSteamID64), kill.Tick, kill.WeaponName}
			kills[k] = append(kills[k], kill)
		}
	}
	for k, candidates := range kills {
		if len(candidates) != 1 || len(fatalDamages[k]) != 1 || candidates[0].HasUtilityImpactData {
			continue
		}
		evidence := fatalDamages[k][0].utilityImpactLifecycle
		if evidence == nil {
			continue
		}
		kill := candidates[0]
		kill.HasUtilityImpactData, kill.IsUtilityImpactKill = true, true
		kill.UtilityImpactSource = utilityImpactSourceLifecycle
		kill.UtilityImpactProjectileID, kill.UtilityImpactEffectTick, kill.UtilityImpactEffect = evidence.projectileID, evidence.effectTick, evidence.effect
	}
}
