package api

import (
	"bytes"
	"math"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/funData"
	dem "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

type pinTestClock struct {
	tick int
}

func newPinTestAnalyzer(t *testing.T) *Analyzer {
	t.Helper()
	p := dem.NewParser(bytes.NewReader(nil))
	t.Cleanup(func() { _ = p.Close() })
	return &Analyzer{parser: p}
}

type pinTestProperty struct {
	value    st.PropertyValue
	handlers []st.PropertyUpdateHandler
}

func (p *pinTestProperty) Value() st.PropertyValue { return p.value }
func (p *pinTestProperty) OnUpdate(h st.PropertyUpdateHandler) {
	p.handlers = append(p.handlers, h)
}
func (p *pinTestProperty) publish(value st.PropertyValue) {
	p.value = value
	for _, h := range p.handlers {
		h(value)
	}
}

type pinTestEntity struct {
	id, serial int
	pin        *pinTestProperty
	onDestroy  []func()
}

func (e *pinTestEntity) ID() int        { return e.id }
func (e *pinTestEntity) SerialNum() int { return e.serial }
func (e *pinTestEntity) Property(name string) *pinTestProperty {
	if name == "m_fPinPullTime" && e.pin != nil {
		return e.pin
	}
	return nil
}
func (e *pinTestEntity) PropertyValue(name string) (st.PropertyValue, bool) {
	if p := e.Property(name); p != nil {
		return p.Value(), true
	}
	return st.PropertyValue{}, false
}
func (e *pinTestEntity) OnDestroy(h func()) { e.onDestroy = append(e.onDestroy, h) }
func (e *pinTestEntity) Destroy() {
	for _, h := range e.onDestroy {
		h()
	}
}
func pinValue(t float32) st.PropertyValue { return st.PropertyValue{Any: t} }
func pinEntity(id, serial int, initial float32) *pinTestEntity {
	return &pinTestEntity{id: id, serial: serial, pin: &pinTestProperty{value: pinValue(initial)}}
}

// v6 entities are concrete; test the production property/destroy callbacks with
// a controlled clock instead of embedding parser or entity internals.
func trackPinTestEntity(a *Analyzer, p *pinTestClock, e *pinTestEntity) {
	key := grenadeWeaponKey{e.id, e.serial}
	e.pin.OnUpdate(a.grenadePinPullHandler(key, e.pin.Value(), func() int { return p.tick }))
	e.OnDestroy(a.grenadePinDestroyedHandler(key))
}

func (e *pinTestEntity) pullTick(a *Analyzer, p *pinTestClock) int {
	return a.observedPinPulledTick(grenadeWeaponKey{e.id, e.serial}, e.pin.Value(), p.tick)
}

func TestGrenadePinPullRecordsObservedDemoTick(t *testing.T) {
	p := &pinTestClock{tick: 100}
	a := newPinTestAnalyzer(t)
	e := pinEntity(5, 1, 0)
	trackPinTestEntity(a, p, e)
	e.pin.publish(e.pin.value) // Source 2 sends an initial property update.
	if got := e.pullTick(a, p); got != 0 {
		t.Fatalf("unprimed weapon has pull tick %d", got)
	}
	p.tick = 4227
	e.pin.publish(pinValue(1085.6216)) // Server seconds have a different epoch.
	p.tick = 4302
	e.pin.publish(e.pin.value) // Repeated network values must not move the pull tick.
	if got := e.pullTick(a, p); got != 4227 {
		t.Fatalf("pull tick = %d, want observed demo tick 4227", got)
	}
	p.tick = 4000 // A stale record after a timeline rewind is not a valid pull.
	if got := e.pullTick(a, p); got != 0 {
		t.Fatalf("future pull tick = %d, want unknown", got)
	}
}

func TestGrenadePinPullInitialStateAndReset(t *testing.T) {
	p := &pinTestClock{tick: 100}
	a := newPinTestAnalyzer(t)
	e := pinEntity(5, 1, 1085)
	trackPinTestEntity(a, p, e)
	e.pin.publish(e.pin.value)
	if got := e.pullTick(a, p); got != 0 {
		t.Fatalf("pre-recording pull must remain unknown, got %d", got)
	}
	for _, invalid := range []float32{0, -1, float32(math.NaN()), float32(math.Inf(1))} {
		e.pin.publish(pinValue(invalid))
		p.tick++
		e.pin.publish(pinValue(1100))
		if got := e.pullTick(a, p); got != p.tick {
			t.Fatalf("new pull after reset = %d, want %d", got, p.tick)
		}
		e.pin.publish(pinValue(invalid))
		if got := e.pullTick(a, p); got != 0 {
			t.Fatalf("invalid/reset property retained pull tick %d", got)
		}
	}
	// A missing initial value must not swallow the first actual pull update.
	e = pinEntity(6, 1, 0)
	e.pin.value = st.PropertyValue{}
	trackPinTestEntity(a, p, e)
	p.tick++
	e.pin.publish(pinValue(1200))
	if got := e.pullTick(a, p); got != p.tick {
		t.Fatalf("pull after uninitialized property = %d, want %d", got, p.tick)
	}
}

func TestGrenadePinPullDestructionAndEntityReuse(t *testing.T) {
	p := &pinTestClock{tick: 50}
	a := newPinTestAnalyzer(t)
	old := pinEntity(5, 1, 0)
	trackPinTestEntity(a, p, old)
	old.pin.publish(pinValue(1085))
	p.tick = 60
	old.Destroy()
	if got := old.pullTick(a, p); got != 50 {
		t.Fatalf("same-frame WeaponFire lost pull tick: %d", got)
	}
	replacement := pinEntity(5, 2, 0)
	trackPinTestEntity(a, p, replacement)
	if got := replacement.pullTick(a, p); got != 0 {
		t.Fatalf("reused entity index inherited old pull tick %d", got)
	}
	replacement.pin.publish(pinValue(1086))
	a.cleanupDestroyedGrenadePins()
	if got := old.pullTick(a, p); got != 0 {
		t.Fatalf("destroyed weapon retained pull tick %d", got)
	}
	if got := replacement.pullTick(a, p); got != 60 {
		t.Fatalf("old cleanup removed replacement pull tick: %d", got)
	}
}

func TestUtilityThrowButtonsInvalidPinFallsBackToLookback(t *testing.T) {
	match := &Match{
		PlayerButtons: []*funData.PlayerButtons{
			{SteamID64: 1, Tick: 100, Buttons: uint64(common.ButtonAttack)},
			{SteamID64: 1, Tick: 110, Buttons: 0},
			{SteamID64: 1, Tick: 120, Buttons: uint64(common.ButtonAttack2)},
		},
	}
	for _, pin := range []int{0, 105, 69479} {
		u := &Utility{Tick: 110, PinPulledTick: pin, ThrowerSteamID64: 1}
		applyUtilityThrowButtonsInWindow(match, u, 110-32)
		if !u.HasAttack || u.HasAttack2 {
			t.Fatalf("pin %d: attack=%v attack2=%v, want true/false", pin, u.HasAttack, u.HasAttack2)
		}
	}
	if got := getThrowButtonState(match, 1, 130, 110); got != (throwButtonState{}) {
		t.Fatalf("invalid window must not reuse a last/future state: %+v", got)
	}
}

func TestGrenadePinPullRoundRestoreDiscardsPreviousPulls(t *testing.T) {
	p := &pinTestClock{tick: 50}
	a := newPinTestAnalyzer(t)
	a.match, a.currentRound = &Match{}, &Round{Number: 2}
	e := pinEntity(5, 1, 0)
	trackPinTestEntity(a, p, e)
	e.pin.publish(pinValue(1085))
	other := pinEntity(6, 1, 0)
	trackPinTestEntity(a, p, other)
	other.pin.publish(pinValue(1086))
	other.Destroy()
	a.resetCurrentRound()
	if got := e.pullTick(a, p); got != 0 || len(a.destroyedGrenadePins) != 0 {
		t.Fatalf("round restore retained pull tick %d or pending destruction", got)
	}
	p.tick = 60
	e.pin.publish(e.pin.value)
	if got := e.pullTick(a, p); got != 0 {
		t.Fatalf("unchanged property resurrected pre-restore pull tick %d", got)
	}
	e.pin.publish(pinValue(1090))
	if got := e.pullTick(a, p); got != 60 {
		t.Fatalf("new post-restore pull tick = %d, want 60", got)
	}
}
