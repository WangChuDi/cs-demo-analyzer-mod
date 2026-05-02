package tests

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api"
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/akiver/cs-demo-analyzer/tests/testsutils"
)

func TestPlayersCSVExport_CounterStrafeColumnsAndTeamDamageOrder(t *testing.T) {
	demoName := "renown_match_8_2025_mirage"
	demoPath := testsutils.GetDemoPath("cs2", demoName)
	outputDir := t.TempDir()

	err := api.AnalyzeAndExportDemo(demoPath, outputDir, api.AnalyzeAndExportDemoOptions{
		Source: constants.DemoSourceRenown,
		Format: constants.ExportFormatCSV,
	})
	if err != nil {
		t.Fatalf("failed to export demo as csv: %v", err)
	}

	playersCSVPath := filepath.Join(outputDir, demoName+"_players.csv")
	file, err := os.Open(playersCSVPath)
	if err != nil {
		t.Fatalf("failed to open exported players csv: %v", err)
	}
	defer file.Close()

	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("failed to read exported players csv: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected exported players csv to contain header and at least one data row")
	}

	header := rows[0]
	indexByHeader := make(map[string]int, len(header))
	for index, column := range header {
		indexByHeader[column] = index
	}

	orderedHeaderBlock := []string{
		"team attack damage",
		"team utility damage",
		"team flash duration",
		"first shot count",
		"first shot hit count",
		"first shot accuracy",
		"counter-strafing success rate",
		"counter-strafing avg delta tick",
		"counter-strafing delta stddev tick",
		"counter-strafing a->d avg delta tick",
		"counter-strafing a->d perfect rate",
		"counter-strafing d->a avg delta tick",
		"counter-strafing d->a perfect rate",
		"counter-strafing w->s avg delta tick",
		"counter-strafing w->s perfect rate",
		"counter-strafing s->w avg delta tick",
		"counter-strafing s->w perfect rate",
		"counter-strafing combo avg delta tick",
		"counter-strafing combo delta stddev tick",
		"counter-strafing combo perfect rate",
		"counter-strafing perfect rate",
	}
	assertOrderedCSVHeaderBlock(t, indexByHeader, orderedHeaderBlock)

	requiredColumns := []string{
		"team attack damage",
		"team utility damage",
		"first shot count",
		"first shot hit count",
		"first shot accuracy",
		"counter-strafing success rate",
		"counter-strafing avg delta tick",
		"counter-strafing delta stddev tick",
		"counter-strafing a->d avg delta tick",
		"counter-strafing a->d perfect rate",
		"counter-strafing d->a avg delta tick",
		"counter-strafing d->a perfect rate",
		"counter-strafing w->s avg delta tick",
		"counter-strafing w->s perfect rate",
		"counter-strafing s->w avg delta tick",
		"counter-strafing s->w perfect rate",
		"counter-strafing combo avg delta tick",
		"counter-strafing combo delta stddev tick",
		"counter-strafing combo perfect rate",
		"counter-strafing perfect rate",
	}
	for _, column := range requiredColumns {
		if _, ok := indexByHeader[column]; !ok {
			t.Fatalf("expected exported players csv to contain column %q", column)
		}
	}

	playerRow := findCSVRowByName(t, rows[1:], header, "whatsnxt")
	assertCSVCellEquals(t, playerRow, indexByHeader, "team attack damage", "0")
	assertCSVCellEquals(t, playerRow, indexByHeader, "team utility damage", "6")
	assertCSVCellEquals(t, playerRow, indexByHeader, "first shot count", "51")
	assertCSVCellEquals(t, playerRow, indexByHeader, "first shot hit count", "6")
	assertCSVCellEquals(t, playerRow, indexByHeader, "first shot accuracy", "11.764706")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing success rate", "31.372551")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing avg delta tick", "4.744681")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing delta stddev tick", "11.629874")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing a->d avg delta tick", "2.080000")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing a->d perfect rate", "84.000000")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing d->a avg delta tick", "6.941176")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing d->a perfect rate", "76.470589")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing w->s avg delta tick", "11.500000")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing w->s perfect rate", "0.000000")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing s->w avg delta tick", "7.000000")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing s->w perfect rate", "0.000000")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing combo avg delta tick", "56.735294")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing combo delta stddev tick", "63.380415")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing combo perfect rate", "11.764706")
	assertCSVCellEquals(t, playerRow, indexByHeader, "counter-strafing perfect rate", "72.340424")
}

func findCSVRowByName(t *testing.T, rows [][]string, header []string, name string) []string {
	t.Helper()

	nameIndex := -1
	for index, column := range header {
		if column == "name" {
			nameIndex = index
			break
		}
	}
	if nameIndex == -1 {
		t.Fatalf("expected csv header to contain name column")
	}

	for _, row := range rows {
		if len(row) <= nameIndex {
			continue
		}
		if row[nameIndex] == name {
			return row
		}
	}

	t.Fatalf("expected to find csv row for player %q", name)
	return nil
}

func assertCSVCellEquals(t *testing.T, row []string, indexByHeader map[string]int, column string, want string) {
	t.Helper()

	index, ok := indexByHeader[column]
	if !ok {
		t.Fatalf("expected header map to contain column %q", column)
	}
	if index >= len(row) {
		t.Fatalf("expected row to contain index %d for column %q", index, column)
	}
	if got := row[index]; got != want {
		t.Fatalf("expected column %q to be %q but got %q", column, want, got)
	}
}

func assertOrderedCSVHeaderBlock(t *testing.T, indexByHeader map[string]int, orderedColumns []string) {
	t.Helper()

	if len(orderedColumns) == 0 {
		return
	}

	previousIndex := -1
	previousColumn := ""
	for _, column := range orderedColumns {
		index, ok := indexByHeader[column]
		if !ok {
			t.Fatalf("expected exported players csv to contain column %q", column)
		}
		if previousIndex != -1 && index != previousIndex+1 {
			t.Fatalf("expected column %q to immediately follow %q in header order", column, previousColumn)
		}
		previousIndex = index
		previousColumn = column
	}
}
