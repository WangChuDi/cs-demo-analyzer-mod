package api

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

func TestMatchCSVMetadataColumnAlignment(t *testing.T) {
	ct, tt := common.TeamCounterTerrorists, common.TeamTerrorists
	match := &Match{
		DemoFileName: "metadata", Duration: 2000 * time.Second,
		NetworkProtocol: 14191, BuildNumber: 10924,
		TeamA: &Team{CurrentSide: &ct}, TeamB: &Team{CurrentSide: &tt},
		DateSource: "file_mtime",
	}
	dir := t.TempDir()
	if err := exportMatchToCSV(match, dir); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "metadata_match.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"duration": "2000.000000", "network protocol": "14191", "build number": "10924",
		"game type": "0", "game mode": "0", "game mode str": "unknown", "is ranked": "0",
		"game mode source": "unknown", "date source": "file_mtime",
		"has game type data": "0", "has game mode data": "0", "footsteps status": "not_observed",
		"aim punch sample count": "0", "bullet damage matched count": "0",
	}
	seen := map[string]bool{}
	for i, name := range rows[0] {
		seen[name] = true
		if expected, ok := want[name]; ok && rows[1][i] != expected {
			t.Errorf("%s = %q, want %q", name, rows[1][i], expected)
		}
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("missing column %q", name)
		}
	}
}

func TestCSVAvailabilityColumnsDistinguishObservedZero(t *testing.T) {
	ct, tt := common.TeamCounterTerrorists, common.TeamTerrorists
	match := &Match{DemoFileName: "availability", TeamA: &Team{CurrentSide: &ct}, TeamB: &Team{CurrentSide: &tt},
		Shots:   []*Shot{{}, {HasAimPunchData: true, HasViewPunchData: true, HasRecoilIndexData: true}},
		Damages: []*Damage{{}, {HasBulletDamageData: true}},
	}
	markHeuristicWallbangDamages(match)
	dir := t.TempDir()
	if err := exportMatchToCSV(match, dir); err != nil {
		t.Fatal(err)
	}
	for table, expected := range map[string]map[string][2]string{
		"shots":   {"has aim punch data": {"0", "1"}, "has view punch data": {"0", "1"}, "has recoil index data": {"0", "1"}},
		"damages": {"has bullet damage data": {"0", "1"}, "num penetrations": {"0", "0"}, "wallbang data source": {"unavailable", "bullet_damage"}},
	} {
		f, err := os.Open(filepath.Join(dir, "availability_"+table+".csv"))
		if err != nil {
			t.Fatal(err)
		}
		rows, err := csv.NewReader(f).ReadAll()
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 3 {
			t.Fatalf("%s: expected two records", table)
		}
		columns := map[string]int{}
		for i, name := range rows[0] {
			columns[name] = i
		}
		for name, values := range expected {
			i, ok := columns[name]
			if !ok {
				t.Fatalf("%s missing column %s", table, name)
			}
			if rows[1][i] != values[0] || rows[2][i] != values[1] {
				t.Fatalf("%s.%s: missing/zero availability incorrect", table, name)
			}
		}
	}
}
