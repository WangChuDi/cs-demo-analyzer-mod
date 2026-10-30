package assertion

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api"
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
)

var update = flag.Bool("update", false, "update snapshots with the current analyzer output")

var (
	uniqueIDPattern     = regexp.MustCompile(`("(?:weaponId|grenadeId|weaponUniqueId)": ")(\d+)(")`)
	projectileIDPattern = regexp.MustCompile(`("projectileId": ")([1-9]\d*)(")`)
	datePattern         = regexp.MustCompile(`("date": ")([^"]*)(")`)
)

// normalizeUniqueIDs makes the random or run-dependent IDs in the JSON stable so the snapshot comparison is
// deterministic.
func normalizeUniqueIDs(jsonBytes []byte) []byte {
	// Weapon and grenade unique IDs are assigned by the parser non-deterministically to fungible items
	// (e.g. a player holding two identical grenades, or a weapon picked up to replace an identical one). The same event
	// can therefore reference a different entity from one run to another, which no test-side aliasing can stabilize, so
	// the IDs are redacted entirely.
	jsonBytes = uniqueIDPattern.ReplaceAll(jsonBytes, []byte("${1}redacted${3}"))

	// Projectile IDs are stable per projectile but their absolute values vary across runs, so replace each one with an
	// alias assigned by order of first appearance. The same projectile keeps the same alias across every record that
	// references it.
	aliases := make(map[string]string)
	return projectileIDPattern.ReplaceAllFunc(jsonBytes, func(m []byte) []byte {
		groups := projectileIDPattern.FindSubmatch(m)
		id := string(groups[2])
		alias, ok := aliases[id]
		if !ok {
			alias = fmt.Sprintf("%d", len(aliases)+1)
			aliases[id] = alias
		}
		out := append([]byte{}, groups[1]...)
		out = append(out, alias...)
		return append(out, groups[3]...)
	})
}

// eventSortKey holds the fields used to deterministically order event slices (shots, grenades) whose order depends on
// parser internals that may change across runs.
type eventSortKey struct {
	Tick       int
	SteamID64  uint64
	WeaponName constants.WeaponName
	X          float64
	Y          float64
	Z          float64
}

func (a eventSortKey) less(b eventSortKey) bool {
	if a.Tick != b.Tick {
		return a.Tick < b.Tick
	}

	if a.SteamID64 != b.SteamID64 {
		return a.SteamID64 < b.SteamID64
	}

	if a.WeaponName != b.WeaponName {
		return a.WeaponName < b.WeaponName
	}

	if a.X != b.X {
		return a.X < b.X
	}

	if a.Y != b.Y {
		return a.Y < b.Y
	}

	return a.Z < b.Z
}

// sortByEventKey stably sorts s by the eventSortKey extracted from each element.
func sortByEventKey[T any](s []T, getKey func(T) eventSortKey) {
	sort.SliceStable(s, func(i, j int) bool {
		return getKey(s[i]).less(getKey(s[j]))
	})
}

// sortUnorderedSlices sorts slices whose order depends on the parser internals Go maps that may change across
// runs so that JSON output is deterministic.
func sortUnorderedSlices(match *api.Match) {
	sort.SliceStable(match.PlayerEconomies, func(i, j int) bool {
		a, b := match.PlayerEconomies[i], match.PlayerEconomies[j]
		if a.RoundNumber != b.RoundNumber {
			return a.RoundNumber < b.RoundNumber
		}

		if a.SteamID64 != b.SteamID64 {
			return a.SteamID64 < b.SteamID64
		}

		// BOTs all share the SteamID64 0, fall back to the name to keep a stable order.
		return a.Name < b.Name
	})

	sortByEventKey(match.Shots, func(s *api.Shot) eventSortKey {
		return eventSortKey{s.Tick, s.PlayerSteamID64, s.WeaponName, s.X, s.Y, s.Z}
	})

	sortByEventKey(match.GrenadeProjectilesDestroy, func(g *api.GrenadeProjectileDestroy) eventSortKey {
		return eventSortKey{g.Tick, g.ThrowerSteamID64, g.GrenadeName, g.X, g.Y, g.Z}
	})

	sortByEventKey(match.GrenadePositions, func(g *api.GrenadePosition) eventSortKey {
		return eventSortKey{g.Tick, g.ThrowerSteamID64, g.GrenadeName, g.X, g.Y, g.Z}
	})
}

// describeFirstLineDiff returns a human-readable description of the first line where "want" and "got" diverge.
func describeFirstLineDiff(want []byte, got []byte) string {
	wantLines := bytes.Split(want, []byte("\n"))
	gotLines := bytes.Split(got, []byte("\n"))

	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		var wantLine, gotLine []byte
		if i < len(wantLines) {
			wantLine = wantLines[i]
		}
		if i < len(gotLines) {
			gotLine = gotLines[i]
		}
		if !bytes.Equal(wantLine, gotLine) {
			return fmt.Sprintf("first difference at line %d:\nwant: %s\ngot:  %s", i+1, wantLine, gotLine)
		}
	}

	return "contents are identical except for line endings or trailing data"
}

// Keep upstream snapshots as the baseline for upstream fields. Fork extensions
// are covered by dedicated metric tests instead of duplicating every snapshot.
// Missing baseline fields and changed array lengths still fail the comparison.
func projectSnapshotFields(got, want any) any {
	switch expected := want.(type) {
	case map[string]any:
		actual, ok := got.(map[string]any)
		if !ok {
			return got
		}
		result := make(map[string]any, len(expected))
		// Numeric keys identify dynamic maps such as players by Steam ID;
		// an unexpected player must remain visible to the comparison.
		for key := range expected {
			if _, err := strconv.ParseUint(key, 10, 64); err == nil {
				for actualKey, value := range actual {
					result[actualKey] = value
				}
				break
			}
		}
		for key, value := range expected {
			if current, exists := actual[key]; exists {
				result[key] = projectSnapshotFields(current, value)
			}
		}
		return result
	case []any:
		actual, ok := got.([]any)
		if !ok || len(actual) != len(expected) {
			return got
		}
		result := make([]any, len(actual))
		for i := range actual {
			result[i] = projectSnapshotFields(actual[i], expected[i])
		}
		return result
	default:
		return got
	}
}

func canonicalSnapshot(data []byte, template []byte) ([]byte, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if template != nil {
		var expected any
		decoder := json.NewDecoder(bytes.NewReader(template))
		decoder.UseNumber()
		if err := decoder.Decode(&expected); err != nil {
			return nil, err
		}
		value = projectSnapshotFields(value, expected)
	}
	return json.MarshalIndent(value, "", "  ")
}

// AssertMatchSnapshot marshals the match to JSON and compares it against the snapshot tests/snapshots/<demoName>.json.
// Run go test ./tests -update to (re)generate snapshots, then review the git diff.
func AssertMatchSnapshot(t *testing.T, match *api.Match, demoName string) {
	t.Helper()

	// The match's date corresponds to the demo file's modification time, which is machine-specific and so unstable across runs.
	// The only exception is with Valve demos that have a .info file that contains the original date.
	// We redact the date if it matches the demo file's mod time.
	shouldRedactDate := false
	if stat, err := os.Stat(match.DemoFilePath); err == nil && match.Date.Equal(stat.ModTime()) {
		shouldRedactDate = true
	}

	// time.Time is marshaled in its local zone, so the rendered string depends on the test machine's
	// timezone (for example +01:00 locally vs Z on CI). Pin it to UTC so it stays stable.
	match.Date = match.Date.UTC()

	// The demo path is machine-specific, exclude it from the comparison.
	match.DemoFilePath = ""
	sortUnorderedSlices(match)

	got, err := json.MarshalIndent(match, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal match to JSON: %v", err)
	}
	if shouldRedactDate {
		got = datePattern.ReplaceAll(got, []byte("${1}redacted${3}"))
	}
	got = append(got, '\n')

	snapshotPath := filepath.Join("snapshots", demoName+".json")
	if *update {
		got = normalizeUniqueIDs(got)
		if err := os.WriteFile(snapshotPath, got, 0o644); err != nil {
			t.Fatalf("failed to write snapshot %s: %v", snapshotPath, err)
		}
		t.Logf("updated snapshot %s", snapshotPath)
		return
	}

	want, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("failed to read snapshot %s, run go test ./tests -run %s -update to create it: %v", snapshotPath, t.Name(), err)
	}
	got, err = canonicalSnapshot(got, want)
	if err != nil {
		t.Fatal(err)
	}
	want, err = canonicalSnapshot(want, nil)
	if err != nil {
		t.Fatal(err)
	}
	got = normalizeUniqueIDs(got)
	want = normalizeUniqueIDs(want)

	if bytes.Equal(got, want) {
		return
	}

	gotPath := snapshotPath + ".got"
	if err := os.WriteFile(gotPath, got, 0o644); err != nil {
		t.Fatalf("failed to write actual output to %s: %v", gotPath, err)
	}
	t.Errorf(
		"match does not match snapshot %s\n%s\nactual output written to %s, diff it against the snapshot or run go test ./tests -run %s -update if the change is expected",
		snapshotPath, describeFirstLineDiff(want, got), gotPath, t.Name(),
	)
}
