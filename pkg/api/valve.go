package api

import (
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

func createValveAnalyzer(analyzer *Analyzer) {
	parser := analyzer.parser
	match := analyzer.match
	analyzer.matchStarted = parser.GameState().IsMatchStarted

	parser.RegisterEventHandler(func(event events.MatchStart) {
		currentRound := analyzer.currentRound
		currentRound.StartTick = analyzer.currentTick()
		analyzer.updateTeamNames()
	})

	parser.RegisterEventHandler(func(event events.MatchStartedChanged) {
		isMatchStarted := !event.OldIsStarted && event.NewIsStarted
		if isMatchStarted {
			currentRound := analyzer.currentRound
			currentRound.TeamASide = *match.TeamA.CurrentSide
			currentRound.TeamBSide = *match.TeamB.CurrentSide
		}
	})

	parser.RegisterEventHandler(func(event events.GameHalfEnded) {
		analyzer.isFirstRoundOfHalf = true
	})

	parser.RegisterEventHandler(analyzer.defaultRoundFreezetimeChangedHandler)

	parser.RegisterEventHandler(analyzer.defaultRoundStartHandler)

	parser.RegisterEventHandler(analyzer.defaultRoundEndOfficiallyHandler)

	parser.RegisterEventHandler(analyzer.defaultAnnouncementWinPanelMatchHandler)
}
