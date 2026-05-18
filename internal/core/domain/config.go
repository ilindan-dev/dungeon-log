package domain

import "time"

// Config represents the parameters of the dungeon challenge.
// It dictates the rules and time boundaries for a successful completion.
type Config struct {
	// Floors defines the total number of regular floors in the dungeon
	// before reaching the boss room.
	Floors int

	// Monsters defines the exact number of monsters that must be defeated
	// on each regular floor to consider it cleared.
	Monsters int

	// OpenAt specifies the exact time of day the dungeon becomes accessible.
	OpenAt time.Time

	// Duration represents the total active time window during which
	// players can complete the challenge.
	Duration time.Duration
}
