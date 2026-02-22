package api

import (
	"fmt"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

type GrenadePosition struct {
	Frame            int                  `json:"frame"`
	Tick             int                  `json:"tick"`
	RoundNumber      int                  `json:"roundNumber"`
	GrenadeID        string               `json:"grenadeId"`
	ProjectileID     int64                `json:"projectileId"`
	X                float64              `json:"x"`
	Y                float64              `json:"y"`
	Z                float64              `json:"z"`
	ThrowerSteamID64 uint64               `json:"throwerSteamId"`
	ThrowerName      string               `json:"throwerName"`
	ThrowerSide      common.Team          `json:"throwerSide"`
	ThrowerTeamName  string               `json:"throwerTeamName"`
	ThrowerVelocityX float64              `json:"throwerVelocityX"`
	ThrowerVelocityY float64              `json:"throwerVelocityY"`
	ThrowerVelocityZ float64              `json:"throwerVelocityZ"`
	ThrowerPitch     float32              `json:"throwerPitch"`
	ThrowerYaw       float32              `json:"throwerYaw"`
	VelocityX        float64              `json:"velocityX"`
	VelocityY        float64              `json:"velocityY"`
	VelocityZ        float64              `json:"velocityZ"`
	GrenadeName      constants.WeaponName `json:"grenadeName"`
}

func newGrenadePositionFromProjectile(analyzer *Analyzer, projectile *common.GrenadeProjectile) *GrenadePosition {
	if projectile.WeaponInstance == nil {
		fmt.Println("Projectile weapon instance nil in grenade projectile position")
		return nil
	}

	thrower := projectile.Thrower
	if thrower == nil {
		fmt.Println("Thrower nil in grenade projectile position, falling back to owner")
		thrower = projectile.WeaponInstance.Owner
		if thrower == nil {
			fmt.Println("Owner nil in grenade projectile position")
			return nil
		}
	}

	velocity := getPlayerVelocity(thrower, analyzer)
	projectileVelocity := getGrenadeProjectileVelocity(projectile)

	parser := analyzer.parser
	throwerTeam := thrower.Team
	return &GrenadePosition{
		Frame:            parser.CurrentFrame(),
		Tick:             analyzer.currentTick(),
		RoundNumber:      analyzer.currentRound.Number,
		GrenadeID:        projectile.WeaponInstance.UniqueID2().String(),
		ProjectileID:     projectile.UniqueID(),
		GrenadeName:      equipmentToWeaponName[projectile.WeaponInstance.Type],
		X:                projectile.Position().X,
		Y:                projectile.Position().Y,
		Z:                projectile.Position().Z,
		ThrowerSteamID64: thrower.SteamID64,
		ThrowerName:      thrower.Name,
		ThrowerSide:      throwerTeam,
		ThrowerTeamName:  analyzer.match.Team(throwerTeam).Name,
		ThrowerVelocityX: velocity.X,
		ThrowerVelocityY: velocity.Y,
		ThrowerVelocityZ: velocity.Z,
		ThrowerYaw:       thrower.ViewDirectionX(),
		ThrowerPitch:     thrower.ViewDirectionY(),
		VelocityX:        projectileVelocity.X,
		VelocityY:        projectileVelocity.Y,
		VelocityZ:        projectileVelocity.Z,
	}
}

func getGrenadeProjectileVelocity(projectile *common.GrenadeProjectile) r3.Vector {
	if projectile == nil || projectile.Entity == nil {
		return r3.Vector{}
	}
	if val, ok := projectile.Entity.PropertyValue("m_vecVelocity"); ok {
		return val.R3Vec()
	}

	trajectory := projectile.Trajectory
	if len(trajectory) < 2 {
		return r3.Vector{}
	}
	last := trajectory[len(trajectory)-1]
	prev := trajectory[len(trajectory)-2]
	seconds := last.Time.Seconds() - prev.Time.Seconds()
	if seconds <= 0 {
		return r3.Vector{}
	}
	return r3.Vector{
		X: (last.Position.X - prev.Position.X) / seconds,
		Y: (last.Position.Y - prev.Position.Y) / seconds,
		Z: (last.Position.Z - prev.Position.Z) / seconds,
	}
}
