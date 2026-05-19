package domain

import "errors"

// Domain-level errors representing business rule violations.
var (
	// ErrPlayerNotRegistered indicates an action was attempted by an unknown player.
	ErrPlayerNotRegistered = errors.New("player is not registered")

	// ErrPlayerDead indicates the player's health has reached zero or below.
	ErrPlayerDead = errors.New("player is dead")

	// ErrDungeonClosed indicates the action occurred after the allowed duration.
	ErrDungeonClosed = errors.New("dungeon opening time has expired")

	// ErrInvalidMove indicates an action that violates the physical rules
	// of the dungeon (e.g., going back a floor).
	ErrInvalidMove = errors.New("impossible move")

	// ErrDisqualified indicates the player has explicitly stated they cannot continue.
	ErrDisqualified = errors.New("player cannot continue")
)
