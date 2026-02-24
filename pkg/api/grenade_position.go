package api

import (
	"fmt"
	"math"

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
	Speed            float64              `json:"speed"`
	GrenadeName      constants.WeaponName `json:"grenadeName"`
}

type grenadeProjectilePositionSample struct {
	position r3.Vector
	tick     int
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
	projectileVelocity := getGrenadeProjectileVelocity(analyzer, projectile)

	parser := analyzer.parser
	throwerTeam := thrower.Team
	speed := vectorSpeed(projectileVelocity)
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
		Speed:            speed,
	}
}

func getGrenadeProjectileVelocity(analyzer *Analyzer, projectile *common.GrenadeProjectile) r3.Vector {
	if analyzer == nil || projectile == nil {
		return r3.Vector{}
	}
	currentTick := analyzer.currentTick()
	currentPos := projectile.Position()
	var velocity r3.Vector
	if previous, ok := analyzer.lastGrenadeProjectilePosition[projectile.UniqueID()]; ok {
		tickDelta := currentTick - previous.tick
		tickTime := analyzer.parser.TickTime().Seconds()
		if tickDelta > 0 && tickTime > 0 {
			seconds := float64(tickDelta) * tickTime
			velocity = r3.Vector{
				X: (currentPos.X - previous.position.X) / seconds,
				Y: (currentPos.Y - previous.position.Y) / seconds,
				Z: (currentPos.Z - previous.position.Z) / seconds,
			}
		}
	}
	if analyzer.lastGrenadeProjectilePosition != nil {
		analyzer.lastGrenadeProjectilePosition[projectile.UniqueID()] = grenadeProjectilePositionSample{
			position: currentPos,
			tick:     currentTick,
		}
	}
	return velocity
}

func vectorSpeed(vector r3.Vector) float64 {
	return math.Sqrt(vector.X*vector.X + vector.Y*vector.Y + vector.Z*vector.Z)
}
