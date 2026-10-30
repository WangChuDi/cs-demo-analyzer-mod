package demo

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"google.golang.org/protobuf/proto"
)

func TestDemoDateProvenanceAndMissingMatchTime(t *testing.T) {
	mtime := time.Unix(1700000000, 0)
	for _, tc := range []struct {
		name   string
		info   *msg.CDataGCCStrike15V2_MatchInfo
		source string
		date   time.Time
	}{
		{"no_info", nil, "file_mtime", mtime},
		{"missing_match_time", &msg.CDataGCCStrike15V2_MatchInfo{}, "file_mtime", mtime},
		{"match_time", &msg.CDataGCCStrike15V2_MatchInfo{Matchtime: proto.Uint32(1600000000)}, "match_info", time.Unix(1600000000, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "date.dem")
			header, err := proto.Marshal(&msg.CDemoFileHeader{DemoFileStamp: proto.String("PBDEMS2"), MapName: proto.String("de_mirage")})
			if err != nil {
				t.Fatal(err)
			}
			data := append([]byte("PBDEMS2\x00"), make([]byte, 8)...)
			data = append(data, 1, 0) // DEM_FileHeader, tick 0
			data = binary.AppendUvarint(data, uint64(len(header)))
			data = append(data, header...)
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(path, mtime, mtime); err != nil {
				t.Fatal(err)
			}
			if tc.info != nil {
				info, err := proto.Marshal(tc.info)
				if err != nil {
					t.Fatal(err)
				}
				// A non-empty proto with no match time exercises the fallback path.
				if len(info) == 0 {
					info = []byte{8, 1}
				}
				if err := os.WriteFile(path+".info", info, 0600); err != nil {
					t.Fatal(err)
				}
			}
			demo, err := GetDemoFromPath(path)
			if err != nil {
				t.Fatal(err)
			}
			if demo.DateSource != tc.source || !demo.Date.Equal(tc.date) {
				t.Fatalf("date=%v source=%s, want %v %s", demo.Date, demo.DateSource, tc.date, tc.source)
			}
		})
	}
}
