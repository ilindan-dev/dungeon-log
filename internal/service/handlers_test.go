package service

import (
	"errors"
	"testing"
	"time"

	"github.com/ilindan-dev/dungeon-log/internal/core/domain"
)

//nolint:gocognit // Table-driven tests inherently have higher length and complexity metrics
func setupTestService() *DungeonService {
	return &DungeonService{
		cfg: &domain.Config{
			Floors:   2,
			Monsters: 2,
		},
		players: make(map[int]*domain.Player),
	}
}

//nolint:gocognit // Table-driven tests inherently have higher length and complexity metrics
func TestHandleRegistrationAndEnter(t *testing.T) {
	s := setupTestService()
	baseTime := time.Now()

	evReg := domain.Event{ID: domain.EventRegistered, PlayerID: 1, Time: baseTime}

	if err := s.handleRegistration(nil, evReg); err != nil {
		t.Fatalf("unexpected error on registration: %v", err)
	}
	if p, exists := s.players[1]; !exists || p.State != domain.StateOutside || len(p.MonstersKilled) != 4 {
		t.Errorf("player not initialized correctly")
	}

	p := s.players[1]

	p.Health = 50
	_ = s.handleRegistration(p, evReg)
	if p.Health != 50 {
		t.Errorf("registration overwritten existing player state")
	}

	evEnter := domain.Event{ID: domain.EventEntered, PlayerID: 1, Time: baseTime}

	if err := s.handleEnter(p, evEnter); err != nil {
		t.Fatalf("unexpected error on enter: %v", err)
	}
	if p.State != domain.StateInDungeon || p.CurrentFloor != 1 {
		t.Errorf("player not entered correctly")
	}

	if err := s.handleEnter(p, evEnter); !errors.Is(err, domain.ErrInvalidMove) {
		t.Errorf("expected ErrInvalidMove when entering twice, got %v", err)
	}
}

//nolint:funlen,gocognit // Table-driven tests inherently have higher length and complexity metrics
func TestFloorAndMonsterMechanics(t *testing.T) {
	s := setupTestService()
	baseTime := time.Now()

	tests := []struct {
		name        string
		handlerName string
		setupPlayer func() *domain.Player
		event       domain.Event
		wantErr     error
		validate    func(*testing.T, *domain.Player)
	}{
		{
			name:        "Prev floor from floor 1 (Edge Case: Boundary)",
			handlerName: "PrevFloor",
			setupPlayer: func() *domain.Player { return &domain.Player{CurrentFloor: 1} },
			event:       domain.Event{Time: baseTime},
			wantErr:     domain.ErrInvalidMove,
			validate:    func(_ *testing.T, _ *domain.Player) {},
		},
		{
			name:        "Prev floor from floor 2 (Valid)",
			handlerName: "PrevFloor",
			setupPlayer: func() *domain.Player { return &domain.Player{CurrentFloor: 2} },
			event:       domain.Event{Time: baseTime},
			wantErr:     nil,
			validate: func(t *testing.T, p *domain.Player) {
				if p.CurrentFloor != 1 {
					t.Errorf("expected floor 1, got %d", p.CurrentFloor)
				}
			},
		},
		{
			name:        "Kill monster on cleared floor (Edge Case: Overkill)",
			handlerName: "KillMonster",
			setupPlayer: func() *domain.Player {
				return &domain.Player{CurrentFloor: 1, MonstersKilled: []int{0, 2, 0, 0}}
			},
			event:   domain.Event{Time: baseTime},
			wantErr: domain.ErrInvalidMove,
			validate: func(t *testing.T, p *domain.Player) {
				if p.MonstersKilled[1] != 2 {
					t.Errorf("killed monster count increased on cleared floor")
				}
			},
		},
		{
			name:        "Kill monster on boss floor (Edge Case: No monsters at boss)",
			handlerName: "KillMonster",
			setupPlayer: func() *domain.Player { return &domain.Player{CurrentFloor: 3} },
			event:       domain.Event{Time: baseTime},
			wantErr:     domain.ErrInvalidMove,
			validate:    func(_ *testing.T, _ *domain.Player) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.setupPlayer()
			if p.MonstersKilled == nil {
				p.MonstersKilled = make([]int, 4)
			}
			if p.FloorClearTimes == nil {
				p.FloorClearTimes = make([]time.Duration, 4)
			}

			var err error
			switch tt.handlerName {
			case "PrevFloor":
				err = s.handlePrevFloor(p, tt.event)
			case "KillMonster":
				err = s.handleKillMonster(p, tt.event)
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("handler() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.validate(t, p)
		})
	}
}

func TestBossAndLeaveMechanics(t *testing.T) {
	s := setupTestService()
	baseTime := time.Now()

	t.Run("Enter Boss - Speedrunner Edge Case", func(t *testing.T) {
		p := &domain.Player{
			CurrentFloor:   2,
			MonstersKilled: []int{0, 2, 1, 0},
		}
		err := s.handleEnteredBoss(p, domain.Event{Time: baseTime})
		if !errors.Is(err, domain.ErrInvalidMove) {
			t.Errorf("speedrunner entered boss without clearing floors, err: %v", err)
		}
	})

	t.Run("Enter Boss - Valid", func(t *testing.T) {
		p := &domain.Player{
			CurrentFloor:    2,
			MonstersKilled:  []int{0, 2, 2, 0},
			FloorClearTimes: make([]time.Duration, 4),
		}
		err := s.handleEnteredBoss(p, domain.Event{Time: baseTime})
		if err != nil {
			t.Errorf("failed to enter boss floor: %v", err)
		}
		if p.CurrentFloor != 3 {
			t.Errorf("expected floor 3, got %d", p.CurrentFloor)
		}
	})

	t.Run("Kill Boss - Twice Edge Case", func(t *testing.T) {
		p := &domain.Player{
			CurrentFloor: 3,
			BossKilled:   true,
		}
		err := s.handleKilledBoss(p, domain.Event{Time: baseTime})
		if !errors.Is(err, domain.ErrInvalidMove) {
			t.Errorf("expected ErrInvalidMove when killing boss twice")
		}
	})

	t.Run("Leave Dungeon - Win vs Fail", func(t *testing.T) {
		pWin := &domain.Player{BossKilled: true, FloorClearTimes: make([]time.Duration, 4)}
		_ = s.handleLeftDungeon(pWin, domain.Event{Time: baseTime})
		if pWin.State != domain.StateSuccess {
			t.Errorf("expected SUCCESS, got %v", pWin.State)
		}

		pFail := &domain.Player{BossKilled: false, FloorClearTimes: make([]time.Duration, 4)}
		_ = s.handleLeftDungeon(pFail, domain.Event{Time: baseTime})
		if pFail.State != domain.StateFail {
			t.Errorf("expected FAIL, got %v", pFail.State)
		}
	})
}

//nolint:funlen,gocognit,gocyclo // Table-driven tests inherently have higher length and complexity metrics
func TestHealthMechanics(t *testing.T) {
	s := setupTestService()
	baseTime := time.Now()

	tests := []struct {
		name        string
		handlerName string
		setupPlayer func() *domain.Player
		eventParam  string
		wantErr     error
		validate    func(*testing.T, *domain.Player)
	}{
		{
			name:        "Damage - Normal",
			handlerName: "Damage",
			setupPlayer: func() *domain.Player { return &domain.Player{Health: 100} },
			eventParam:  "30",
			wantErr:     nil,
			validate: func(t *testing.T, p *domain.Player) {
				if p.Health != 70 {
					t.Errorf("expected HP 70, got %d", p.Health)
				}
			},
		},
		{
			name:        "Damage - Lethal (Underflow protection)",
			handlerName: "Damage",
			setupPlayer: func() *domain.Player {
				return &domain.Player{Health: 10, FloorClearTimes: make([]time.Duration, 4)}
			},
			eventParam: "50",
			wantErr:    domain.ErrPlayerDead,
			validate: func(t *testing.T, p *domain.Player) {
				if p.Health != 0 {
					t.Errorf("expected HP 0 on death, got %d", p.Health)
				}
				if p.DeathTime != baseTime {
					t.Errorf("death time not set correctly")
				}
			},
		},
		{
			name:        "Damage - Invalid Log format (Ignore)",
			handlerName: "Damage",
			setupPlayer: func() *domain.Player { return &domain.Player{Health: 50} },
			eventParam:  "not_a_number",
			wantErr:     nil,
			validate: func(t *testing.T, p *domain.Player) {
				if p.Health != 50 {
					t.Errorf("health changed on invalid damage log")
				}
			},
		},
		{
			name:        "Heal - Normal",
			handlerName: "Heal",
			setupPlayer: func() *domain.Player { return &domain.Player{Health: 50} },
			eventParam:  "20",
			wantErr:     nil,
			validate: func(t *testing.T, p *domain.Player) {
				if p.Health != 70 {
					t.Errorf("expected HP 70, got %d", p.Health)
				}
			},
		},
		{
			name:        "Heal - Overheal (Cap at 100)",
			handlerName: "Heal",
			setupPlayer: func() *domain.Player { return &domain.Player{Health: 80} },
			eventParam:  "50",
			wantErr:     nil,
			validate: func(t *testing.T, p *domain.Player) {
				if p.Health != 100 {
					t.Errorf("expected HP 100 after overheal, got %d", p.Health)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.setupPlayer()
			ev := domain.Event{Time: baseTime, ExtraParam: tt.eventParam}

			var err error
			switch tt.handlerName {
			case "Damage":
				err = s.handleDamage(p, ev)
			case "Heal":
				err = s.handleHeal(p, ev)
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("handler() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.validate(t, p)
		})
	}
}
