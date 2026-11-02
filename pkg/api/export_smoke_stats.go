package api

import (
	"encoding/json"

	"github.com/akiver/cs-demo-analyzer/internal/converters"
	"github.com/akiver/cs-demo-analyzer/internal/csv"
)

func writeSmokeStatsCSV(match *Match, outputPath string) {
	f, i := converters.Float64ToString, converters.IntToString
	u, b := converters.Uint64ToString, converters.BoolToString
	clouds := [][]string{{"id", "round number", "start tick", "ready tick", "end tick", "end reason", "x", "y", "z", "model", "voxel status", "voxel updates", "voxel frames", "voxel unavailable updates", "voxel first tick", "voxel last tick", "voxel unavailable reasons", "match checksum"}}
	for _, cloud := range match.SmokeClouds {
		reasons := ""
		if len(cloud.VoxelUnavailableReasons) > 0 {
			data, _ := json.Marshal(cloud.VoxelUnavailableReasons)
			reasons = string(data)
		}
		clouds = append(clouds, []string{cloud.ID, i(cloud.RoundNumber), i(cloud.StartTick), f(cloud.ReadyTick), i(cloud.EndTick), cloud.EndReason, f(cloud.X), f(cloud.Y), f(cloud.Z), smokeStatsModel,
			cloud.VoxelStatus, i(cloud.VoxelUpdates), i(cloud.VoxelFrames), i(cloud.VoxelUnavailableUpdates), i(cloud.VoxelFirstTick), i(cloud.VoxelLastTick), reasons, match.Checksum})
	}
	csv.WriteLinesIntoCsvFile(outputPath+"_smoke_clouds.csv", clouds)
	visits := [][]string{{"model", "round number", "steamid", "player name", "side", "smoke id", "entry tick", "exit tick", "entry x", "entry y", "entry z", "exit x", "exit y", "exit z", "entry reason", "end reason", "duration", "min center distance", "is full pass", "has assistance data", "is dry pass", "reason", "assistance", "match checksum"}}
	for _, visit := range match.SmokeVisits {
		assistance := ""
		if len(visit.Assistance) > 0 {
			data, _ := json.Marshal(visit.Assistance)
			assistance = string(data)
		}
		visits = append(visits, []string{visit.Model, i(visit.RoundNumber), u(visit.PlayerSteamID64), visit.PlayerName, converters.TeamToString(visit.Side), visit.SmokeID,
			f(visit.EntryTick), f(visit.ExitTick), f(visit.EntryPosition.X), f(visit.EntryPosition.Y), f(visit.EntryPosition.Z), f(visit.ExitPosition.X), f(visit.ExitPosition.Y), f(visit.ExitPosition.Z),
			visit.EntryReason, visit.EndReason, f(visit.Duration), f(visit.MinCenterDistance), b(visit.IsFullPass), b(visit.HasAssistanceData), b(visit.IsDryPass), visit.Reason, assistance, match.Checksum})
	}
	csv.WriteLinesIntoCsvFile(outputPath+"_smoke_visits.csv", visits)
	voxelVisits := [][]string{{"model", "round number", "steamid", "player name", "smoke ids", "entry tick", "exit tick", "entry x", "entry y", "entry z", "exit x", "exit y", "exit z", "entry reason", "end reason", "duration", "density threshold", "match checksum"}}
	for _, visit := range match.SmokeVoxelVisits {
		ids, _ := json.Marshal(visit.SmokeIDs)
		voxelVisits = append(voxelVisits, []string{visit.Model, i(visit.RoundNumber), u(visit.PlayerSteamID64), visit.PlayerName, string(ids), f(visit.EntryTick), f(visit.ExitTick),
			f(visit.EntryPosition.X), f(visit.EntryPosition.Y), f(visit.EntryPosition.Z), f(visit.ExitPosition.X), f(visit.ExitPosition.Y), f(visit.ExitPosition.Z),
			visit.EntryReason, visit.EndReason, f(visit.Duration), f(float64(visit.DensityThreshold)), match.Checksum})
	}
	csv.WriteLinesIntoCsvFile(outputPath+"_smoke_voxel_visits.csv", voxelVisits)
}
