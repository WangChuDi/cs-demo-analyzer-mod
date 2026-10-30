package api

import (
	"math"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// Entity indices are reused; the serial number identifies the weapon instance.
type grenadeWeaponKey struct {
	entityID  int
	serialNum int
}

type grenadePinPull struct {
	time float32
	tick int
}

func (analyzer *Analyzer) registerUtilityPinHandlers() {
	parser := analyzer.parser
	parser.RegisterEventHandler(func(events.DataTablesParsed) {
		for _, name := range []string{
			"CFlashbang", "CHEGrenade", "CSmokeGrenade", "CDecoyGrenade", "CMolotovGrenade", "CIncendiaryGrenade",
		} {
			if class := parser.ServerClasses().FindByName(name); class != nil {
				class.OnEntityCreated(analyzer.trackGrenadePinPull)
			}
		}
	})
	parser.RegisterEventHandler(func(events.FrameDone) {
		analyzer.cleanupDestroyedGrenadePins()
	})
}

func grenadePinPullTime(value st.PropertyValue) float32 {
	if value.S2 && value.Any == nil {
		return 0
	}
	t := value.Float()
	if t <= 0 || math.IsNaN(float64(t)) || math.IsInf(float64(t), 0) {
		return 0
	}
	return t
}

func (analyzer *Analyzer) trackGrenadePinPull(entity st.Entity) {
	property := entity.Property("m_fPinPullTime")
	if property == nil {
		return
	}
	key := grenadeWeaponKey{entity.ID(), entity.SerialNum()}
	// OnUpdate also sends the initial value. A grenade already primed when it
	// enters the recording has no observed pull tick and must remain unknown (0).
	previousTime := grenadePinPullTime(property.Value())
	property.OnUpdate(func(value st.PropertyValue) {
		pinTime := grenadePinPullTime(value)
		if pinTime == previousTime {
			return
		}
		previousTime = pinTime
		delete(analyzer.grenadePinPulls, key)
		if pinTime == 0 {
			return
		}
		tick := analyzer.currentTick()
		if tick <= 0 {
			return
		}
		if analyzer.grenadePinPulls == nil {
			analyzer.grenadePinPulls = make(map[grenadeWeaponKey]grenadePinPull)
		}
		analyzer.grenadePinPulls[key] = grenadePinPull{time: pinTime, tick: tick}
	})
	entity.OnDestroy(func() {
		// CS2 can destroy the weapon before dispatching its WeaponFire in the
		// same frame. Keep the pull tick until those events have been processed.
		analyzer.destroyedGrenadePins = append(analyzer.destroyedGrenadePins, key)
	})
}

func (analyzer *Analyzer) cleanupDestroyedGrenadePins() {
	for _, key := range analyzer.destroyedGrenadePins {
		delete(analyzer.grenadePinPulls, key)
	}
	analyzer.destroyedGrenadePins = analyzer.destroyedGrenadePins[:0]
}

func (analyzer *Analyzer) resetGrenadePins() {
	clear(analyzer.grenadePinPulls)
	analyzer.destroyedGrenadePins = nil
}

func getPinPulledTick(analyzer *Analyzer, weaponEntity st.Entity) int {
	if analyzer == nil || weaponEntity == nil {
		return 0
	}
	pull, ok := analyzer.grenadePinPulls[grenadeWeaponKey{weaponEntity.ID(), weaponEntity.SerialNum()}]
	if !ok || pull.tick <= 0 || pull.tick > analyzer.currentTick() {
		return 0
	}
	value, ok := weaponEntity.PropertyValue("m_fPinPullTime")
	if !ok || grenadePinPullTime(value) != pull.time {
		return 0
	}
	return pull.tick
}
