package service

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ilindan-dev/dungeon-log/internal/core/domain"
)

// Mocks

// mockParser simulates reading events from a file.
type mockParser struct {
	events []domain.Event
	idx    int
	err    error
}

func (m *mockParser) Next() (domain.Event, error) {
	if m.err != nil {
		return domain.Event{}, m.err
	}
	if m.idx >= len(m.events) {
		return domain.Event{}, io.EOF
	}
	ev := m.events[m.idx]
	m.idx++
	return ev, nil
}

// mockReporter records emitted events and the final report for assertions.
type mockReporter struct {
	incomingCount int
	outgoing      []domain.EventID
	finalPlayers  []*domain.Player
}

func (m *mockReporter) EmitIncoming(_ domain.Event) error {
	m.incomingCount++
	return nil
}

func (m *mockReporter) EmitOutgoing(_ time.Time, id domain.EventID, _ int, _ string) error {
	m.outgoing = append(m.outgoing, id)
	return nil
}

func (m *mockReporter) PrintFinalReport(players []*domain.Player) error {
	m.finalPlayers = players
	return nil
}

// Tests

func TestNewDungeonService(t *testing.T) {
	cfg := &domain.Config{}
	svc := NewDungeonService(cfg, &mockParser{}, &mockReporter{})

	if svc == nil {
		t.Fatal("NewDungeonService returned nil")
	}
	if svc.players == nil {
		t.Fatal("players map not initialized")
	}
}

//nolint:funlen,gocognit // Table-driven tests are naturally long
func TestDispatch(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC)
	cfg := &domain.Config{
		OpenAt:   baseTime,
		Duration: 1 * time.Hour,
		Floors:   2,
	}

	tests := []struct {
		name         string
		setupPlayer  func() *domain.Player
		event        domain.Event
		wantState    domain.State
		wantOutgoing domain.EventID // 0 if no outgoing expected
	}{
		{
			name: "Deadline Exceeded (Fail Active Player)",
			setupPlayer: func() *domain.Player {
				return &domain.Player{ID: 1, State: domain.StateInDungeon}
			},
			event: domain.Event{
				ID: domain.EventKilledMonster, PlayerID: 1,
				Time: baseTime.Add(61 * time.Minute),
			},
			wantState:    domain.StateFail,
			wantOutgoing: 0,
		},
		{
			name:         "Unregistered Player Action (Disqualify)",
			setupPlayer:  func() *domain.Player { return nil },
			event:        domain.Event{ID: domain.EventEntered, PlayerID: 2, Time: baseTime.Add(10 * time.Minute)},
			wantState:    domain.StateDisqual,
			wantOutgoing: domain.OutEventDisqualified,
		},
		{
			name: "Ghost Action (Ignore Terminal State)",
			setupPlayer: func() *domain.Player {
				return &domain.Player{ID: 3, State: domain.StateFail}
			},
			event: domain.Event{
				ID: domain.EventKilledMonster, PlayerID: 3,
				Time: baseTime.Add(10 * time.Minute),
			},
			wantState:    domain.StateFail,
			wantOutgoing: 0,
		},
		{
			name: "Invalid Move Mapping",
			setupPlayer: func() *domain.Player {
				return &domain.Player{ID: 4, State: domain.StateInDungeon, CurrentFloor: 1}
			},
			event:        domain.Event{ID: domain.EventPrevFloor, PlayerID: 4, Time: baseTime.Add(10 * time.Minute)},
			wantState:    domain.StateInDungeon,
			wantOutgoing: domain.OutEventInvalidMove,
		},
		{
			name: "Death Mapping",
			setupPlayer: func() *domain.Player {
				return &domain.Player{
					ID: 5, State: domain.StateInDungeon,
					Health: 10, FloorClearTimes: make([]time.Duration, 4),
				}
			},

			event: domain.Event{
				ID: domain.EventReceivedDamage, PlayerID: 5,
				Time: baseTime.Add(10 * time.Minute), ExtraParam: "50",
			},
			wantState:    domain.StateFail,
			wantOutgoing: domain.OutEventDead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := &mockReporter{}
			svc := NewDungeonService(cfg, &mockParser{}, reporter)

			p := tt.setupPlayer()
			pID := tt.event.PlayerID
			if p != nil {
				svc.players[pID] = p
			}

			svc.dispatch(tt.event)

			gotPlayer := svc.players[pID]
			if gotPlayer == nil {
				t.Fatalf("player missing from map after dispatch")
			}
			if gotPlayer.State != tt.wantState {
				t.Errorf("expected state %v, got %v", tt.wantState, gotPlayer.State)
			}

			if tt.wantOutgoing != 0 {
				if len(reporter.outgoing) == 0 || reporter.outgoing[0] != tt.wantOutgoing {
					t.Errorf("expected outgoing event %v, got %v", tt.wantOutgoing, reporter.outgoing)
				}
			} else if len(reporter.outgoing) > 0 {
				t.Errorf("expected no outgoing events, but got %v", reporter.outgoing)
			}
		})
	}
}

func TestRun(t *testing.T) {
	cfg := &domain.Config{
		OpenAt:   time.Now(),
		Duration: 1 * time.Hour,
		Floors:   2,
	}

	t.Run("Process to EOF and Finalize", func(t *testing.T) {
		parser := &mockParser{
			events: []domain.Event{
				{ID: domain.EventRegistered, PlayerID: 1},
				{ID: domain.EventRegistered, PlayerID: 2},
				{ID: domain.EventEntered, PlayerID: 2},
			},
		}
		reporter := &mockReporter{}
		svc := NewDungeonService(cfg, parser, reporter)

		err := svc.Run()
		if err != nil {
			t.Fatalf("unexpected Run() error: %v", err)
		}

		if reporter.finalPlayers == nil {
			t.Fatalf("PrintFinalReport was not called")
		}

		for _, p := range svc.players {
			if p.State != domain.StateFail {
				t.Errorf("player %d state = %v, expected FAIL after EOF finalization", p.ID, p.State)
			}
		}
	})

	t.Run("Parser Error", func(t *testing.T) {
		expectedErr := errors.New("parser broke")
		parser := &mockParser{err: expectedErr}
		svc := NewDungeonService(cfg, parser, &mockReporter{})

		err := svc.Run()
		if err == nil {
			t.Fatalf("expected error from Run(), got nil")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})
}
