package api

import "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"

// Demo ticks can start long after the server starts. Use the explicit network
// clock for networked GameTick_t properties; never substitute a demo tick.
func (analyzer *Analyzer) updateServerTick(message *msg.CNETMsg_Tick) {
	if message.Tick != nil {
		analyzer.serverTick, analyzer.hasServerTick = int(message.GetTick()), true
	}
}

func (analyzer *Analyzer) updateSnapshotTick(message *msg.CSVCMsg_PacketEntities) {
	if message.ServerTick != nil {
		analyzer.serverTick, analyzer.hasServerTick = int(message.GetServerTick()), true
	}
}
