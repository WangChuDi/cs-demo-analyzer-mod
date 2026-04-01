package constants

type HeuristicWallbangWeaponDamageModel struct {
	BaseDamage     float64
	RangeModifier  float64
	ArmorRatio     float64
	HeadMultiplier float64
}

const (
	// These thresholds are used by the wallbang heuristic path because demos/parser signals often do not expose
	// direct non-lethal penetration damage information reliably.
	HeuristicWallbangMaxShotFrameDistance        = 48
	HeuristicWallbangMaxVictimPositionFrameDelta = 48
	HeuristicWallbangHelmetSnapshotFrameWindow   = 8
	HeuristicWallbangMinDelta                    = 1.0
	HeuristicWallbangMinLossRatio                = 0.0
	HeuristicWallbangIncludeHeadshots            = true
)

var HeuristicWallbangWeaponDamageModels = map[WeaponName]HeuristicWallbangWeaponDamageModel{
	WeaponAK47:    {BaseDamage: 36, RangeModifier: 0.98, ArmorRatio: 1.55, HeadMultiplier: 4.0},
	WeaponM4A4:    {BaseDamage: 33, RangeModifier: 0.97, ArmorRatio: 1.40, HeadMultiplier: 4.0},
	WeaponM4A1:    {BaseDamage: 38, RangeModifier: 0.94, ArmorRatio: 1.40, HeadMultiplier: 3.475},
	WeaponAWP:     {BaseDamage: 115, RangeModifier: 0.99, ArmorRatio: 1.95, HeadMultiplier: 4.0},
	WeaponDeagle:  {BaseDamage: 53, RangeModifier: 0.85, ArmorRatio: 1.864, HeadMultiplier: 3.9},
	WeaponUSP:     {BaseDamage: 35, RangeModifier: 0.91, ArmorRatio: 1.01, HeadMultiplier: 4.0},
	WeaponP250:    {BaseDamage: 38, RangeModifier: 0.90, ArmorRatio: 1.28, HeadMultiplier: 4.0},
	WeaponGlock:   {BaseDamage: 30, RangeModifier: 0.85, ArmorRatio: 0.94, HeadMultiplier: 4.0},
	WeaponFamas:   {BaseDamage: 30, RangeModifier: 0.96, ArmorRatio: 1.40, HeadMultiplier: 4.0},
	WeaponGalilAR: {BaseDamage: 30, RangeModifier: 0.98, ArmorRatio: 1.55, HeadMultiplier: 4.0},
}
