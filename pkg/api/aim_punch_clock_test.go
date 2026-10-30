package api

import (
	"math"
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
	"google.golang.org/protobuf/proto"
)

func TestAimPunchGameTickDecodesSignedNetworkTicks(t *testing.T) {
	for _, test := range []struct {
		name  string
		wire  any
		want  int
		valid bool
	}{
		{"cache first recoil", uint32(135382), 67691, true},
		{"cache final recoil", uint32(387076), 193538, true},
		{"zero", uint32(0), 0, true},
		{"negative sentinel", uint32(1), -1, true},
		{"minimum int32", uint32(math.MaxUint32), math.MinInt32, true},
		{"maximum int32", uint32(math.MaxUint32 - 1), math.MaxInt32, true},
		{"already decoded positive", int32(67691), 67691, true},
		{"already decoded negative", int32(-1), -1, true},
		{"missing", nil, 0, false},
		{"unsupported type", float32(1), 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, valid := aimPunchGameTick(st.PropertyValue{Any: test.wire})
			if got != test.want || valid != test.valid {
				t.Fatalf("got (%d, %v), want (%d, %v)", got, valid, test.want, test.valid)
			}
		})
	}
}

func TestAimPunchServerClockRequiresExplicitNetworkTick(t *testing.T) {
	a := &Analyzer{}
	a.updateServerTick(&msg.CNETMsg_Tick{})
	a.updateSnapshotTick(&msg.CSVCMsg_PacketEntities{})
	if a.hasServerTick {
		t.Fatal("absent clock must remain unavailable")
	}
	a.updateServerTick(&msg.CNETMsg_Tick{Tick: proto.Uint32(0)})
	if !a.hasServerTick || a.serverTick != 0 {
		t.Fatal("explicit zero is a valid server tick")
	}
	a.updateServerTick(&msg.CNETMsg_Tick{Tick: proto.Uint32(67467)})
	a.updateSnapshotTick(&msg.CSVCMsg_PacketEntities{})
	if a.serverTick != 67467 {
		t.Fatal("missing snapshot tick replaced the observed server clock")
	}
	a.updateSnapshotTick(&msg.CSVCMsg_PacketEntities{ServerTick: proto.Uint32(67468)})
	if a.serverTick != 67468 {
		t.Fatal("explicit snapshot tick was not observed")
	}
}
