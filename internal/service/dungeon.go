package service

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"time"

	"github.com/ilindan-dev/dungeon-log/internal/core/domain"
	"github.com/ilindan-dev/dungeon-log/internal/core/ports"
)

// DungeonService acts as the central orchestrator and state machine for the challenge.
// It maintains the volatile state of all participating players in memory and
// evaluates incoming events against the rules defined in the configuration.
type DungeonService struct {
	cfg      *domain.Config
	parser   ports.EventParser
	reporter ports.Reporter
	players  map[int]*domain.Player
}

// NewDungeonService constructs a new DungeonService with the provided configuration
// and port implementations (Dependency Injection). It initializes the internal
// memory structures required to track participant progress.
func NewDungeonService(cfg *domain.Config, p ports.EventParser, r ports.Reporter) *DungeonService {
	return &DungeonService{
		cfg:      cfg,
		parser:   p,
		reporter: r,
		players:  make(map[int]*domain.Player),
	}
}

// Run starts the main event loop, processing logs until EOF,
// and then generates the final report.
func (s *DungeonService) Run() error {
	for {
		event, err := s.parser.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to read next event: %w", err)
		}
		s.dispatch(event)
	}

	var finalPlayers []*domain.Player
	for _, p := range s.players {
		if p.State == domain.StateOutside || p.State == domain.StateInDungeon {
			p.State = domain.StateFail
		}

		// Вся сложная математика теперь изолирована
		s.calculateFinalMetrics(p)

		finalPlayers = append(finalPlayers, p)
	}

	slices.SortFunc(finalPlayers, func(a, b *domain.Player) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return s.reporter.PrintFinalReport(finalPlayers)
}

// dispatch routes the incoming event to the appropriate state handler.
//
//nolint:gocyclo // State machine orchestrators inherently have high cyclomatic complexity
func (s *DungeonService) dispatch(e domain.Event) {
	deadline := s.cfg.OpenAt.Add(s.cfg.Duration)
	if e.Time.After(deadline) {
		p, exists := s.players[e.PlayerID]
		if exists && (p.State == domain.StateInDungeon || p.State == domain.StateOutside) {
			p.State = domain.StateFail
		}
		return
	}

	player := s.players[e.PlayerID]

	if player == nil {
		if e.ID == domain.EventRegistered {
			_ = s.reporter.EmitIncoming(e)
			_ = s.handleRegistration(nil, e)
		} else {
			s.players[e.PlayerID] = &domain.Player{
				ID:              e.PlayerID,
				State:           domain.StateDisqual,
				Health:          100,
				MonstersKilled:  make([]int, s.cfg.Floors+2),
				FloorClearTimes: make([]time.Duration, s.cfg.Floors+2),
			}
			_ = s.reporter.EmitOutgoing(e.Time, domain.OutEventDisqualified, e.PlayerID, "")
		}
		return
	}

	if isTerminalState(player.State) {
		return
	}

	err := s.handleStateTransition(player, e)

	if err == nil || errors.Is(err, domain.ErrPlayerDead) {
		_ = s.reporter.EmitIncoming(e)
	}

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidMove):
			_ = s.reporter.EmitOutgoing(e.Time, domain.OutEventInvalidMove, e.PlayerID, strconv.Itoa(int(e.ID)))
		case errors.Is(err, domain.ErrPlayerDead):
			_ = s.reporter.EmitOutgoing(e.Time, domain.OutEventDead, e.PlayerID, "")
			player.State = domain.StateFail
		case errors.Is(err, domain.ErrDisqualified):
			_ = s.reporter.EmitOutgoing(e.Time, domain.OutEventDisqualified, e.PlayerID, "")
			player.State = domain.StateDisqual
		}
	}
}

// calculateFinalMetrics computes the total time spent and average floor clear time
// for a player at the end of the trial.
func (s *DungeonService) calculateFinalMetrics(p *domain.Player) {
	if !p.EnterTime.IsZero() {
		switch {
		case p.State == domain.StateFail && !p.DeathTime.IsZero():
			p.TimeSpent = p.DeathTime.Sub(p.EnterTime)
		case p.LeaveTime.IsZero():
			deadline := s.cfg.OpenAt.Add(s.cfg.Duration)
			p.TimeSpent = deadline.Sub(p.EnterTime)
		default:
			p.TimeSpent = p.LeaveTime.Sub(p.EnterTime)
		}
	}

	var totalFloorTime time.Duration
	var clearedFloors int
	regularFloors := s.cfg.Floors - 1

	if regularFloors > 0 && len(p.FloorClearTimes) > regularFloors {
		for i := 1; i <= regularFloors; i++ {
			if s.isFloorCleared(p, i) {
				totalFloorTime += p.FloorClearTimes[i]
				clearedFloors++
			}
		}
		if clearedFloors > 0 {
			p.AvgFloorClearTime = totalFloorTime / time.Duration(clearedFloors)
		} else {
			p.AvgFloorClearTime = 0
		}
	}
}
