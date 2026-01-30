package funData

import (
	"github.com/akiver/cs-demo-analyzer/pkg/api"
)

// FakePlayerLeechFeed 用于测试 Leech/Feed 功能的玩家模拟数据
type FakePlayerLeechFeed struct {
	SteamID64  uint64
	Name       string
	LeechValue int
	LeechCount int
	FeedValue  int
	FeedCount  int
}

// AssertPlayerLeechFeed 验证玩家的 Leech/Feed 数据
func AssertPlayerLeechFeed(match *api.Match, expected FakePlayerLeechFeed) (passed bool, errMsg string) {
	player := match.PlayersBySteamID[expected.SteamID64]
	if player == nil {
		return false, "player not found with SteamID " + string(rune(expected.SteamID64))
	}

	if player.LeechValue != expected.LeechValue {
		return false, formatError("LeechValue", expected.Name, expected.LeechValue, player.LeechValue)
	}
	if player.LeechCount != expected.LeechCount {
		return false, formatError("LeechCount", expected.Name, expected.LeechCount, player.LeechCount)
	}
	if player.FeedValue != expected.FeedValue {
		return false, formatError("FeedValue", expected.Name, expected.FeedValue, player.FeedValue)
	}
	if player.FeedCount != expected.FeedCount {
		return false, formatError("FeedCount", expected.Name, expected.FeedCount, player.FeedCount)
	}

	return true, ""
}

func formatError(field, playerName string, expected, actual int) string {
	return "expected player " + playerName + " " + field + " to be different, got " + string(rune(actual))
}
