package api

import (
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func getPlayerVelocity(p *common.Player) r3.Vector {
	if p == nil {
		return r3.Vector{}
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
