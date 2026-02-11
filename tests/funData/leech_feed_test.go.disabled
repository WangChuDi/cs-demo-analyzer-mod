package funData

import (
	"fmt"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api"
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
)

// Test_LeechFeed_MatchZy_Bleed_vs_Parivision tests the Leech/Feed functionality
// using a known MatchZy demo file.
func Test_LeechFeed_MatchZy_Bleed_vs_Parivision(t *testing.T) {
	demoName := "match730_003690733799101956373_0521179063_234"
	// testsutils.GetDemoPath assumes the test is running from "tests/" directory (../cs-demos)
	// Since we are in "tests/funData/", we need to go up one more level (../../cs-demos)
	demoPath := "../../cs-demos/cs2/" + demoName + ".dem"
	match, err := api.AnalyzeDemo(demoPath, api.AnalyzeDemoOptions{
		Source: constants.DemoSourceMatchZy,
	})
	if err != nil {
		t.Error(err)
		return
	}

	// 打印所有玩家的 Leech/Feed 数据用于分析
	t.Log("=== Leech/Feed 统计 ===")
	for steamID, player := range match.PlayersBySteamID {
		if player.LeechValue > 0 || player.FeedValue > 0 {
			t.Logf("玩家: %s (SteamID: %d)", player.Name, steamID)
			t.Logf("  LeechValue: %d, LeechCount: %d", player.LeechValue, player.LeechCount)
			t.Logf("  FeedValue: %d, FeedCount: %d", player.FeedValue, player.FeedCount)
		}
	}

	// 验证 Leech/Feed 数据的基本一致性
	// 整场比赛中，总 LeechValue 应等于总 FeedValue
	// 总 LeechCount 应等于总 FeedCount
	var totalLeechValue, totalFeedValue int
	var totalLeechCount, totalFeedCount int
	for _, player := range match.PlayersBySteamID {
		totalLeechValue += player.LeechValue
		totalFeedValue += player.FeedValue
		totalLeechCount += player.LeechCount
		totalFeedCount += player.FeedCount
	}

	if totalLeechValue != totalFeedValue {
		t.Errorf("Leech/Feed 数值不一致: 总 LeechValue (%d) != 总 FeedValue (%d)", totalLeechValue, totalFeedValue)
	}
	if totalLeechCount != totalFeedCount {
		t.Errorf("Leech/Feed 计数不一致: 总 LeechCount (%d) != 总 FeedCount (%d)", totalLeechCount, totalFeedCount)
	}

	t.Logf("=== 汇总 ===")
	t.Logf("总 LeechValue: %d, 总 FeedValue: %d", totalLeechValue, totalFeedValue)
	t.Logf("总 LeechCount: %d, 总 FeedCount: %d", totalLeechCount, totalFeedCount)
}

// Test_LeechFeed_PrintAllPlayers 打印所有玩家的 Leech/Feed 统计，用于调试和分析
func Test_LeechFeed_PrintAllPlayers(t *testing.T) {
	testCases := []struct {
		gameFolder string
		demoName   string
		source     constants.DemoSource
	}{
		{"cs2", "match730_003690733799101956373_0521179063_234", constants.DemoSourceMatchZy},
	}

	for _, tc := range testCases {
		t.Run(tc.demoName, func(t *testing.T) {
			// testsutils.GetDemoPath assumes the test is running from "tests/" directory (../cs-demos)
			// Since we are in "tests/funData/", we need to go up one more level (../../cs-demos)
			demoPath := "../../cs-demos/" + tc.gameFolder + "/" + tc.demoName + ".dem"
			match, err := api.AnalyzeDemo(demoPath, api.AnalyzeDemoOptions{
				Source: tc.source,
			})
			if err != nil {
				t.Skipf("跳过测试 (无法加载 demo): %v", err)
				return
			}

			fmt.Printf("\n=== %s Leech/Feed 统计 ===\n", tc.demoName)
			hasAnyLeechFeed := false
			for _, player := range match.PlayersBySteamID {
				if player.LeechValue > 0 || player.FeedValue > 0 {
					hasAnyLeechFeed = true
					fmt.Printf("玩家: %-20s | Leech: 价值=%5d 次数=%d | Feed: 价值=%5d 次数=%d\n",
						player.Name,
						player.LeechValue, player.LeechCount,
						player.FeedValue, player.FeedCount)
				}
			}
			if !hasAnyLeechFeed {
				fmt.Println("该 demo 中没有检测到武器交换事件")
			}
		})
	}
}
