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
		"game type": "0", "game mode": "0", "game mode str": "casual", "is ranked": "0",
	}
	for i, name := range rows[0] {
		if expected, ok := want[name]; ok && rows[1][i] != expected {
			t.Errorf("%s = %q, want %q", name, rows[1][i], expected)
		}
	}
}
