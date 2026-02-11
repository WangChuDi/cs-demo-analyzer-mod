package api

import (
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func getPlayerVelocity(p *common.Player, analyzer *Analyzer) r3.Vector {
	if p == nil {
		return r3.Vector{}
	}

	if lastPos, ok := analyzer.match.lastPlayersPosition[p.SteamID64]; ok {
		tickTime := analyzer.parser.TickTime().Seconds()
		if tickTime > 0 {
			currentPos := p.Position()
			return r3.Vector{
				X: (currentPos.X - lastPos.X) / tickTime,
				Y: (currentPos.Y - lastPos.Y) / tickTime,
				Z: (currentPos.Z - lastPos.Z) / tickTime,
			}
		}
	}

	pawn := p.PlayerPawnEntity()
	if pawn != nil {
		if val, ok := pawn.PropertyValue("m_vecVelocity"); ok {
			return val.R3Vec()
		}
	}
	// Try entity if pawn failed or for Source 1
	if p.Entity != nil {
		if val, ok := p.Entity.PropertyValue("m_vecVelocity"); ok {
			return val.R3Vec()
		}
	}
	return r3.Vector{}
}

func getPlayerPositionEyes(p *common.Player) r3.Vector {
	if p == nil {
		return r3.Vector{}
	}
	pos := p.Position()

	var offset r3.Vector
	pawn := p.PlayerPawnEntity()
	if pawn != nil {
		if val, ok := pawn.PropertyValue("m_vecViewOffset"); ok {
			offset = val.R3Vec()
		}
	} else if p.Entity != nil {
		if val, ok := p.Entity.PropertyValue("m_vecViewOffset"); ok {
			offset = val.R3Vec()
		}
	}

	return pos.Add(offset)
}
