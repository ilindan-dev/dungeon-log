package domain

import "time"

// State represents the current standing of a player in the challenge.
type State string

const (
	StateOutside   State = "OUTSIDE"
	StateInDungeon State = "IN_DUNGEON"
	StateSuccess   State = "SUCCESS"
	StateFail      State = "FAIL"
	StateDisqual   State = "DISQUAL"
)

// Player encapsulates the state, health, and progression metrics of a participant
// navigating the dungeon.
type Player struct {
	ID    int
	State State

	// Health represents the player's current hit points.
	// It is capped at 100 and cannot drop below 0.
	Health int

	// CurrentFloor indicates where the player is currently located.
	// 0 implies the player is outside or in the lobby.
	CurrentFloor int

	// MonstersKilled tracks the number of defeated monsters per floor.
	// The slice index corresponds to the floor number (1-based, index 0 is unused).
	MonstersKilled []int

	// Progression timestamps
	EnterTime time.Time
	LeaveTime time.Time
	DeathTime time.Time

	// Floor tracking metrics
	FloorEnterTime  time.Time
	FloorClearTimes []time.Duration

	// Boss encounter metrics
	BossEnterTime time.Time
	BossKilled    bool
	BossKillTime  time.Duration
}

// IsDead returns true if the player's health has dropped to zero or below.
func (p *Player) IsDead() bool {
	return p.Health <= 0
}
