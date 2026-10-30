package cli

import "testing"

func TestExternalMatchDate(t *testing.T) {
	t.Setenv("CSDA_MATCH_DATE", "2026-10-29T00:00:00+08:00")
	var args cliArgs
	base := []string{"-demo-path", "sample.dem", "-output", "out"}
	if err := args.fromArgs(base); err != nil || args.matchDate != "2026-10-29T00:00:00+08:00" {
		t.Fatalf("environment date: %s / %v", args.matchDate, err)
	}
	if err := args.fromArgs(append(base, "-match-date", "2026-10-28T12:00:00Z")); err != nil || args.matchDate != "2026-10-28T12:00:00Z" {
		t.Fatalf("explicit override: %s / %v", args.matchDate, err)
	}
	for _, bad := range []string{"not-a-date", "2026-10-29", "2026-10-29T00:00:00", "0001-01-01T00:00:00Z"} {
		if err := args.fromArgs(append(base, "-match-date", bad)); err == nil {
			t.Fatalf("accepted invalid timestamp %s", bad)
		}
	}
}
