package service

import (
	"testing"
	"time"

	"github.com/ilindan-dev/dungeon-log/internal/core/domain"
)

func TestIsTerminalState(t *testing.T) {
	tests := []struct {
		name     string
		state    domain.State
		expected bool
	}{
		{"Success is terminal", domain.StateSuccess, true},
		{"Fail is terminal", domain.StateFail, true},
		{"Disqual is terminal", domain.StateDisqual, true},
		{"Outside is not terminal", domain.StateOutside, false},
		{"In dungeon is not terminal", domain.StateInDungeon, false},
		{"Unknown state is not terminal", domain.State("UNKNOWN_MAGIC"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTerminalState(tt.state); got != tt.expected {
				t.Errorf("isTerminalState() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsFloorCleared(t *testing.T) {
	cfg := &domain.Config{Floors: 3, Monsters: 3}
	s := &DungeonService{cfg: cfg}

	tests := []struct {
		name   string
		floor  int
		killed []int
		want   bool
	}{
		{"Lobby (floor 0) is always cleared", 0, []int{0, 0, 0, 0}, true},
		{"Negative floor is always cleared", -1, []int{0, 0, 0, 0}, true},
		{"Boss floor (floor 3) is always cleared", 3, []int{0, 0, 0, 0}, true},
		{"Way past boss floor is always cleared", 5, []int{0, 0, 0, 0}, true},

		{"Floor 1 not cleared (0 killed)", 1, []int{0, 0, 0, 0}, false},
		{"Floor 1 not cleared (1 killed)", 1, []int{0, 1, 0, 0}, false},
		{"Floor 1 cleared exactly (3 killed)", 1, []int{0, 3, 0, 0}, true},

		{"Floor 1 cleared with overkill (4 killed)", 1, []int{0, 4, 0, 0}, true},

		{"Floor 2 checks its own index", 2, []int{0, 3, 0, 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := &domain.Player{MonstersKilled: tt.killed}
			if got := s.isFloorCleared(player, tt.floor); got != tt.want {
				t.Errorf("isFloorCleared() = %v, want %v", got, tt.want)
			}
		})
	}
}

//nolint:funlen,gocognit // Table-driven tests inherently have higher length and complexity metrics
func TestUpdateCurrentFloorTime(t *testing.T) {
	cfg := &domain.Config{Floors: 3, Monsters: 2}
	s := &DungeonService{cfg: cfg}

	baseTime := time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC)
	eventTime := baseTime.Add(5 * time.Minute)

	tests := []struct {
		name           string
		currentFloor   int
		monstersKilled []int
		initialTimes   []time.Duration
		expectedTimes  []time.Duration
		expectedAnchor time.Time
	}{
		{
			name:           "Add time to dirty regular floor",
			currentFloor:   1,
			monstersKilled: []int{0, 0, 0, 0},
			initialTimes:   []time.Duration{0, 1 * time.Minute, 0, 0},
			expectedTimes:  []time.Duration{0, 6 * time.Minute, 0, 0},
			expectedAnchor: eventTime,
		},
		{
			name:           "Do not add time to cleared floor",
			currentFloor:   1,
			monstersKilled: []int{0, 2, 0, 0},
			initialTimes:   []time.Duration{0, 1 * time.Minute, 0, 0},
			expectedTimes:  []time.Duration{0, 1 * time.Minute, 0, 0},
			expectedAnchor: eventTime,
		},
		{
			name:           "Do not add time in lobby (floor 0)",
			currentFloor:   0,
			monstersKilled: []int{0, 0, 0, 0},
			initialTimes:   []time.Duration{0, 0, 0, 0},
			expectedTimes:  []time.Duration{0, 0, 0, 0},
			expectedAnchor: eventTime,
		},
		{
			name:           "Do not add time in boss floor (floor 3)",
			currentFloor:   3,
			monstersKilled: []int{0, 2, 2, 0},
			initialTimes:   []time.Duration{0, 0, 0, 0},
			expectedTimes:  []time.Duration{0, 0, 0, 0},
			expectedAnchor: eventTime,
		},
		{
			name:           "Zero time elapsed edge case",
			currentFloor:   1,
			monstersKilled: []int{0, 0, 0, 0},
			initialTimes:   []time.Duration{0, 0, 0, 0},
			expectedTimes:  []time.Duration{0, 0, 0, 0},
			expectedAnchor: baseTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			playerTimes := make([]time.Duration, len(tt.initialTimes))
			copy(playerTimes, tt.initialTimes)

			playerKilled := make([]int, len(tt.monstersKilled))
			copy(playerKilled, tt.monstersKilled)

			p := &domain.Player{
				CurrentFloor:    tt.currentFloor,
				MonstersKilled:  playerKilled,
				FloorClearTimes: playerTimes,
				FloorEnterTime:  baseTime,
			}

			testEventTime := eventTime
			if tt.name == "Zero time elapsed edge case" {
				testEventTime = baseTime
			}
			s.updateCurrentFloorTime(p, testEventTime)

			if !p.FloorEnterTime.Equal(tt.expectedAnchor) {
				t.Errorf("FloorEnterTime = %v, want %v", p.FloorEnterTime, tt.expectedAnchor)
			}

			for i, gotTime := range p.FloorClearTimes {
				if gotTime != tt.expectedTimes[i] {
					t.Errorf("FloorClearTimes[%d] = %v, want %v", i, gotTime, tt.expectedTimes[i])
				}
			}
		})
	}
}
