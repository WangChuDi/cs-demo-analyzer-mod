package constants

type HeuristicWallbangWeaponDamageModel struct {
	BaseDamage     float64
	RangeModifier  float64
	ArmorRatio     float64
	HeadMultiplier float64
}

const (
	// Retained for existing first-shot attribution. Wallbang inference requires
	// a unique shot in the same frame instead of this wider window.
	HeuristicDamageAttributionMaxShotFrameDistance = 48
)

// Single-projectile guns only. Shotguns require pellet-level damage attribution.
// Damage parameters were compared across every weapons.vdata revision from
// protocol 14005 through 14192. Only MP7/MP5-SD changed (protocol 14130).
// Pinned latest data:
// https://github.com/SteamTracking/GameTracking-CS2/blob/04d3e4a67ac60168d0e79594c69a2116bb3b7278/game/csgo/pak01_dir/scripts/weapons.vdata
// Pinned pre-buff data:
// https://github.com/SteamTracking/GameTracking-CS2/blob/80cd467f43571e979899b3dece753315ab9aeca8/game/csgo/pak01_dir/scripts/weapons.vdata
var HeuristicWallbangWeaponDamageModels = map[WeaponName]HeuristicWallbangWeaponDamageModel{
	WeaponAK47:         {BaseDamage: 36, RangeModifier: 0.98, ArmorRatio: 1.55, HeadMultiplier: 4.0},
	WeaponM4A4:         {BaseDamage: 33, RangeModifier: 0.97, ArmorRatio: 1.40, HeadMultiplier: 4.0},
	WeaponM4A1:         {BaseDamage: 38, RangeModifier: 0.94, ArmorRatio: 1.40, HeadMultiplier: 3.475},
	WeaponAWP:          {BaseDamage: 115, RangeModifier: 0.99, ArmorRatio: 1.95, HeadMultiplier: 4.0},
	WeaponDeagle:       {BaseDamage: 53, RangeModifier: 0.85, ArmorRatio: 1.864, HeadMultiplier: 3.9},
	WeaponUSP:          {BaseDamage: 35, RangeModifier: 0.91, ArmorRatio: 1.01, HeadMultiplier: 4.0},
	WeaponP250:         {BaseDamage: 38, RangeModifier: 0.90, ArmorRatio: 1.28, HeadMultiplier: 4.0},
	WeaponGlock:        {BaseDamage: 30, RangeModifier: 0.85, ArmorRatio: 0.94, HeadMultiplier: 4.0},
	WeaponFamas:        {BaseDamage: 30, RangeModifier: 0.96, ArmorRatio: 1.40, HeadMultiplier: 4.0},
	WeaponGalilAR:      {BaseDamage: 30, RangeModifier: 0.98, ArmorRatio: 1.55, HeadMultiplier: 4.0},
	WeaponAUG:          {BaseDamage: 28, RangeModifier: 0.98, ArmorRatio: 1.80, HeadMultiplier: 4.0},
	WeaponCZ75:         {BaseDamage: 31, RangeModifier: 0.85, ArmorRatio: 1.553, HeadMultiplier: 4.0},
	WeaponDualBerettas: {BaseDamage: 38, RangeModifier: 0.79, ArmorRatio: 1.15, HeadMultiplier: 4.0},
	WeaponFiveSeven:    {BaseDamage: 32, RangeModifier: 0.81, ArmorRatio: 1.823, HeadMultiplier: 4.0},
	WeaponG3SG1:        {BaseDamage: 80, RangeModifier: 0.98, ArmorRatio: 1.65, HeadMultiplier: 4.0},
	WeaponM249:         {BaseDamage: 32, RangeModifier: 0.97, ArmorRatio: 1.60, HeadMultiplier: 4.0},
	WeaponMac10:        {BaseDamage: 29, RangeModifier: 0.80, ArmorRatio: 1.15, HeadMultiplier: 4.0},
	WeaponMP5:          {BaseDamage: 28, RangeModifier: 0.87, ArmorRatio: 1.25, HeadMultiplier: 4.0},
	WeaponMP7:          {BaseDamage: 30, RangeModifier: 0.87, ArmorRatio: 1.25, HeadMultiplier: 4.0},
	WeaponMP9:          {BaseDamage: 26, RangeModifier: 0.87, ArmorRatio: 1.20, HeadMultiplier: 4.0},
	WeaponNegev:        {BaseDamage: 35, RangeModifier: 0.97, ArmorRatio: 1.42, HeadMultiplier: 4.0},
	WeaponP2000:        {BaseDamage: 35, RangeModifier: 0.91, ArmorRatio: 1.01, HeadMultiplier: 4.0},
	WeaponP90:          {BaseDamage: 26, RangeModifier: 0.86, ArmorRatio: 1.38, HeadMultiplier: 4.0},
	WeaponPPBizon:      {BaseDamage: 27, RangeModifier: 0.80, ArmorRatio: 1.26, HeadMultiplier: 4.0},
	WeaponRevolver:     {BaseDamage: 86, RangeModifier: 0.94, ArmorRatio: 1.864, HeadMultiplier: 4.0},
	WeaponScar20:       {BaseDamage: 80, RangeModifier: 0.98, ArmorRatio: 1.65, HeadMultiplier: 4.0},
	WeaponScout:        {BaseDamage: 88, RangeModifier: 0.98, ArmorRatio: 1.70, HeadMultiplier: 4.0},
	WeaponSG553:        {BaseDamage: 30, RangeModifier: 0.98, ArmorRatio: 2.0, HeadMultiplier: 4.0},
	WeaponTec9:         {BaseDamage: 33, RangeModifier: 0.79, ArmorRatio: 1.812, HeadMultiplier: 4.0},
	WeaponUMP45:        {BaseDamage: 35, RangeModifier: 0.75, ArmorRatio: 1.30, HeadMultiplier: 4.0},
}

func HeuristicWallbangModel(weapon WeaponName, protocol int) (HeuristicWallbangWeaponDamageModel, string, bool) {
	model, exists := HeuristicWallbangWeaponDamageModels[weapon]
	if !exists || protocol < 14005 || protocol > 14192 {
		return HeuristicWallbangWeaponDamageModel{}, "", false
	}
	if protocol < 14130 {
		if weapon == WeaponMP7 {
			model.BaseDamage, model.RangeModifier = 29, 0.85
		} else if weapon == WeaponMP5 {
			model.BaseDamage, model.RangeModifier = 27, 0.85
		}
		return model, "cs2_weapons_14005", true
	}
	return model, "cs2_weapons_14130", true
}
