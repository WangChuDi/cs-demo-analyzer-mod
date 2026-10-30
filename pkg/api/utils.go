package api

import (
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
)

// The synchronized parser resolves networked velocity properties and derives
// velocity from snapshot server ticks for older demos. Eager tracking is enabled
// when constructing the parser so grenade creation also sees the prior snapshot.
func getPlayerVelocity(p *common.Player, analyzer *Analyzer) r3.Vector {
	if p == nil {
		return r3.Vector{}
	}

	return p.Velocity()
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
