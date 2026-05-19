package service

import (
	"strconv"
	"time"

	"github.com/ilindan-dev/dungeon-log/internal/core/domain"
)

// handleStateTransition executes the domain logic for a specific event type.
//
//nolint:gocyclo // A single switch router is the most readable way to handle flat state transitions,
func (s *DungeonService) handleStateTransition(p *domain.Player, e domain.Event) error {
	//nolint:exhaustive // Deliberately ignores system-only outgoing events.
	switch e.ID {
	case domain.EventRegistered:
		return s.handleRegistration(p, e)
	case domain.EventEntered:
		if p.State != domain.StateOutside {
			return domain.ErrInvalidMove
		}
		return s.handleEnter(p, e)
	case domain.EventCannotContinue:
		return domain.ErrDisqualified
	}

	if p.State != domain.StateInDungeon {
		return domain.ErrInvalidMove
	}

	switch e.ID {
	case domain.EventKilledMonster:
		return s.handleKillMonster(p, e)
	case domain.EventNextFloor:
		return s.handleNextFloor(p, e)
	case domain.EventPrevFloor:
		return s.handlePrevFloor(p, e)
	case domain.EventEnteredBoss:
		return s.handleEnteredBoss(p, e)
	case domain.EventKilledBoss:
		return s.handleKilledBoss(p, e)
	case domain.EventLeftDungeon:
		return s.handleLeftDungeon(p, e)
	case domain.EventReceivedDamage:
		return s.handleDamage(p, e)
	case domain.EventRestoredHealth:
		return s.handleHeal(p, e)
	default:
		return nil
	}
}

// handleRegistration initializes a new player in the system.
// If the player is already registered, the event is safely ignored
// to prevent overwriting existing progression.
func (s *DungeonService) handleRegistration(p *domain.Player, e domain.Event) error {
	if p != nil {
		return nil
	}
	s.players[e.PlayerID] = &domain.Player{
		ID:              e.PlayerID,
		State:           domain.StateOutside,
		Health:          100,
		MonstersKilled:  make([]int, s.cfg.Floors+2),
		FloorClearTimes: make([]time.Duration, s.cfg.Floors+2),
	}
	return nil
}

// handleEnter transitions a registered player into the dungeon.
// It returns domain.ErrInvalidMove if the player is already inside or has
// reached a terminal state.
func (s *DungeonService) handleEnter(p *domain.Player, e domain.Event) error {
	if p.State != domain.StateOutside {
		return domain.ErrInvalidMove
	}
	p.State = domain.StateInDungeon
	p.CurrentFloor = 1
	p.EnterTime = e.Time
	p.FloorEnterTime = e.Time
	return nil
}

// handleNextFloor advances the player to the next floor and updates the
// active time spent on the current floor before transitioning.
func (s *DungeonService) handleNextFloor(p *domain.Player, e domain.Event) error {
	s.updateCurrentFloorTime(p, e.Time)
	p.CurrentFloor++
	return nil
}

// handlePrevFloor moves the player down one floor.
// It enforces the boundary rule, returning domain.ErrInvalidMove if the player
// attempts to go below the first floor.
func (s *DungeonService) handlePrevFloor(p *domain.Player, e domain.Event) error {
	if p.CurrentFloor <= 1 {
		return domain.ErrInvalidMove
	}
	s.updateCurrentFloorTime(p, e.Time)
	p.CurrentFloor--
	return nil
}

// handleKillMonster records a monster defeat.
// It prevents overkill by returning domain.ErrInvalidMove if the floor is
// already cleared, and forbids killing monsters in the boss room.
func (s *DungeonService) handleKillMonster(p *domain.Player, e domain.Event) error {
	if p.CurrentFloor >= s.cfg.Floors {
		return domain.ErrInvalidMove
	}
	if s.isFloorCleared(p, p.CurrentFloor) {
		return domain.ErrInvalidMove
	}

	s.updateCurrentFloorTime(p, e.Time)
	p.MonstersKilled[p.CurrentFloor]++
	return nil
}

// handleEnteredBoss transitions the player to the final boss room.
// It acts as a strict gatekeeper, ensuring all previous floors are fully
// cleared of monsters before allowing entry.
func (s *DungeonService) handleEnteredBoss(p *domain.Player, e domain.Event) error {
	for i := 1; i < s.cfg.Floors; i++ {
		if !s.isFloorCleared(p, i) {
			return domain.ErrInvalidMove
		}
	}

	s.updateCurrentFloorTime(p, e.Time)
	p.CurrentFloor = s.cfg.Floors
	p.BossEnterTime = e.Time
	return nil
}

// handleKilledBoss registers the defeat of the dungeon's final boss.
// It ensures this action only occurs on the designated boss floor and prevents
// duplicate kills.
func (s *DungeonService) handleKilledBoss(p *domain.Player, e domain.Event) error {
	if p.CurrentFloor != s.cfg.Floors {
		return domain.ErrInvalidMove
	}
	if p.BossKilled {
		return domain.ErrInvalidMove
	}

	p.BossKilled = true
	p.BossKillTime = e.Time.Sub(p.BossEnterTime)
	return nil
}

// handleLeftDungeon marks the player's departure and evaluates the final outcome.
// The trial is marked as SUCCESS only if the boss was defeated; otherwise,
// it results in a FAIL.
func (s *DungeonService) handleLeftDungeon(p *domain.Player, e domain.Event) error {
	s.updateCurrentFloorTime(p, e.Time)
	p.LeaveTime = e.Time

	if p.BossKilled {
		p.State = domain.StateSuccess
	} else {
		p.State = domain.StateFail
	}
	return nil
}

// handleDamage applies damage to the player's health.
// If health drops to zero or below, it caps the value at zero, records the time
// of death, and returns domain.ErrPlayerDead to trigger a system broadcast.
func (s *DungeonService) handleDamage(p *domain.Player, e domain.Event) error {
	damage, err := strconv.Atoi(e.ExtraParam)
	if err != nil {
		//nolint:nilerr // We intentionally ignore broken logs so as not to interrupt the entire cycle.
		return nil
	}

	p.Health -= damage
	if p.IsDead() {
		p.Health = 0
		p.DeathTime = e.Time
		s.updateCurrentFloorTime(p, e.Time)
		return domain.ErrPlayerDead
	}
	return nil
}

// handleHeal restores the player's health.
// It enforces the maximum health cap of 100, preventing any overheal exploits.
func (s *DungeonService) handleHeal(p *domain.Player, e domain.Event) error {
	heal, err := strconv.Atoi(e.ExtraParam)
	if err != nil {
		//nolint:nilerr // We intentionally ignore broken logs so as not to interrupt the entire cycle.
		return nil
	}

	p.Health += heal
	if p.Health > 100 {
		p.Health = 100
	}
	return nil
}
