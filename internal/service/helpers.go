package service

import (
	"time"

	"github.com/ilindan-dev/dungeon-log/internal/core/domain"
)

// isTerminalState checks terminal player state
func isTerminalState(st domain.State) bool {
	return st == domain.StateFail || st == domain.StateDisqual || st == domain.StateSuccess
}

// isFloorCleared dynamically checks if all monsters on a specific floor are dead.
func (s *DungeonService) isFloorCleared(p *domain.Player, floor int) bool {
	if floor < 1 || floor >= s.cfg.Floors {
		return true
	}
	return p.MonstersKilled[floor] >= s.cfg.Monsters
}

// updateCurrentFloorTime adds the time spent since the last anchor (FloorEnterTime)
// to the current floor's total, BUT only if the floor is not yet cleared.
func (s *DungeonService) updateCurrentFloorTime(p *domain.Player, t time.Time) {
	if p.CurrentFloor >= 1 && p.CurrentFloor < s.cfg.Floors {
		if !s.isFloorCleared(p, p.CurrentFloor) {
			spent := t.Sub(p.FloorEnterTime)
			p.FloorClearTimes[p.CurrentFloor] += spent
		}
	}
	p.FloorEnterTime = t
}
