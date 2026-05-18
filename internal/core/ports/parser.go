package ports

import "github.com/ilindan-dev/dungeon-log/internal/core/domain"

// EventParser defines the contract for reading the sequence of dungeon events.
// It abstracts away the source of the events (e.g., file, stdin).
type EventParser interface {
	// Next reads and returns the next chronological event.
	// It should return an error like io.EOF when no more events are available.
	Next() (domain.Event, error)
}
