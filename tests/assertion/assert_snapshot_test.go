package assertion

import (
	"bytes"
	"testing"
)

func TestSnapshotProjectionPreservesCoreRegressions(t *testing.T) {
	want := []byte(`{"players":{"76561198998266210":{"kills":12}},"shots":[{"tick":10}]}`)
	baseline, err := canonicalSnapshot(want, nil)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, got string
		equal     bool
	}{
		{"extension", `{"players":{"76561198998266210":{"kills":12,"leechValue":500}},"shots":[{"tick":10,"speed":20}],"footsteps":[]}`, true},
		{"changed statistic", `{"players":{"76561198998266210":{"kills":11}},"shots":[{"tick":10}]}`, false},
		{"missing field", `{"players":{"76561198998266210":{}},"shots":[{"tick":10}]}`, false},
		{"unexpected player", `{"players":{"76561198998266210":{"kills":12},"76561198998266211":{"kills":0}},"shots":[{"tick":10}]}`, false},
		{"missing event", `{"players":{"76561198998266210":{"kills":12}},"shots":[]}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := canonicalSnapshot([]byte(tc.got), want)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(got, baseline) != tc.equal {
				t.Fatalf("unexpected comparison: %s", got)
			}
		})
	}
}
