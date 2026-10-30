package api

import (
	"encoding/binary"
	"strings"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

type soundFootstep struct {
	entityID int
	name     string
}

// Actual event names from game_sounds_footsteps.vsndevts, not arbitrary sound
// hashes. Land_*, Gear.JumpLand.* and weapon sounds are deliberately excluded.
// https://github.com/SteamTracking/GameTracking-CS2/blob/master/game/csgo/pak01_dir/soundevents/game_sounds_footsteps.vsndevts
var footstepSoundNames = func() map[uint32]string {
	surfaces := strings.Fields(`Concrete Ceiling_Tile Default Wood Wood_Solid SolidMetal ChainLink MetalGrate Ladder MetalVent MetalVehicle MetalBarrel Snow Dirt Mud Slime Grass Tile Water Wood_Basket Glass Gravel Flesh Sand GlassBottle Drywall Cardboard Plastic_Barrel Plastic_Box Rubber Grenade WoodBox Wood_Crate Wood_Panel Weapon Foliage Carpet SlipperySlime Silent WetTile Wood_Ladder Wet_Sand Plastic_milkCrate Plastic_autoCover MetalRailing WetConcrete WetMud`)
	names := make(map[uint32]string, len(surfaces)*2)
	for _, prefix := range []string{"CT_", "T_"} {
		for _, surface := range surfaces {
			name := prefix + surface + ".StepLeft"
			names[soundNameHash(name)] = name
		}
	}
	return names
}()

// Source 2 hashes lowercase sound event names with MurmurHash2 / SRC2 seed.
func soundNameHash(name string) uint32 {
	data := []byte(strings.ToLower(name))
	const multiplier uint32 = 0x5bd1e995
	h := uint32(0x53524332) ^ uint32(len(data))
	for len(data) >= 4 {
		k := binary.LittleEndian.Uint32(data)
		k *= multiplier
		k ^= k >> 24
		k *= multiplier
		h = h*multiplier ^ k
		data = data[4:]
	}
	if len(data) >= 3 {
		h ^= uint32(data[2]) << 16
	}
	if len(data) >= 2 {
		h ^= uint32(data[1]) << 8
	}
	if len(data) >= 1 {
		h ^= uint32(data[0])
		h *= multiplier
	}
	h ^= h >> 13
	h *= multiplier
	h ^= h >> 15
	return h
}

func (analyzer *Analyzer) registerSoundFootstepHandler() {
	analyzer.parser.RegisterNetMessageHandler(func(event *msg.CMsgSosStartSoundEvent) {
		name, known := footstepSoundNames[event.GetSoundeventHash()]
		if !known || event.GetSourceEntityIndex() <= 0 || !analyzer.matchStarted() {
			return
		}
		analyzer.pendingSoundFootsteps = append(analyzer.pendingSoundFootsteps, soundFootstep{
			entityID: int(event.GetSourceEntityIndex()), name: name})
	})
}

func (analyzer *Analyzer) flushFootsteps() {
	// Game events take precedence when a recording supplies both representations.
	seen := map[*common.Player]bool{}
	appendFootstep := func(player *common.Player, source, name string) {
		if player == nil || seen[player] {
			return
		}
		seen[player] = true
		velocity := getPlayerVelocity(player, analyzer)
		footstep := newFootstep(analyzer, events.Footstep{Player: player}, velocity.X, velocity.Y, velocity.Z)
		footstep.DataSource, footstep.SoundEventName = source, name
		analyzer.match.Footsteps = append(analyzer.match.Footsteps, footstep)
	}
	for _, event := range analyzer.pendingFootsteps {
		appendFootstep(event.Player, "game_event", "")
	}
	if len(analyzer.pendingSoundFootsteps) > 0 {
		// Sounds refer to pawn entity IDs, not controller IDs or player slots.
		players := map[int]*common.Player{}
		for _, player := range analyzer.parser.GameState().Participants().Playing() {
			if pawn := player.PlayerPawnEntity(); pawn != nil {
				players[pawn.ID()] = player
			}
		}
		for _, sound := range analyzer.pendingSoundFootsteps {
			appendFootstep(players[sound.entityID], "sound_event", sound.name)
		}
	}
	analyzer.pendingFootsteps = nil
	analyzer.pendingSoundFootsteps = nil
}
