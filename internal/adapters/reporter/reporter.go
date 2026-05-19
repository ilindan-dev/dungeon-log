package reporter

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ilindan-dev/dungeon-log/internal/core/domain"
)

// BaseReporter implements ports.Reporter and writes to any io.Writer.
type BaseReporter struct {
	out    io.Writer
	closer io.Closer
}

// NewStdoutReporter creates a reporter that writes to os.Stdout.
func NewStdoutReporter() *BaseReporter {
	return &BaseReporter{
		out:    os.Stdout,
		closer: nil,
	}
}

// NewFileReporter creates a reporter that writes to a specified file path.
// It opens the file for writing, creating it if necessary, and truncating it if it exists.
func NewFileReporter(path string) (*BaseReporter, error) {
	//nolint:gosec // This is a CLI tool, receiving file paths from the user is intended behavior
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open report file: %w", err)
	}

	return &BaseReporter{
		out:    file,
		closer: file,
	}, nil
}

// Close ensures the file descriptor is released if a file is used.
func (r *BaseReporter) Close() error {
	if r.closer != nil {
		return r.closer.Close()
	}
	return nil
}

// EmitIncoming formats and prints a valid action taken by a player.
func (r *BaseReporter) EmitIncoming(e domain.Event) error {
	timeStr := e.Time.Format("15:04:05")
	msg := formatIncomingEvent(e)
	_, err := fmt.Fprintf(r.out, "[%s] %s\n", timeStr, msg)
	return err
}

// EmitOutgoing formats and prints a system-generated response.
func (r *BaseReporter) EmitOutgoing(t time.Time, eventID domain.EventID, playerID int, extra string) error {
	timeStr := t.Format("15:04:05")
	var msg string

	switch eventID {
	case domain.OutEventDisqualified:
		msg = fmt.Sprintf("Player [%d] is disqualified", playerID)
	case domain.OutEventDead:
		msg = fmt.Sprintf("Player [%d] is dead", playerID)
	case domain.OutEventInvalidMove:
		msg = fmt.Sprintf("Player [%d] makes imposible move [%s]", playerID, extra) // As it was in the task
	}

	_, err := fmt.Fprintf(r.out, "[%s] %s\n", timeStr, msg)
	return err
}

// PrintFinalReport calculates statistics and outputs the final summary for all players.
func (r *BaseReporter) PrintFinalReport(players []*domain.Player) error {
	if _, err := fmt.Fprintln(r.out, "Final report:"); err != nil {
		return err
	}

	for _, p := range players {
		timeSpent := p.LeaveTime.Sub(p.EnterTime)
		if p.State == domain.StateFail && !p.DeathTime.IsZero() {
			timeSpent = p.DeathTime.Sub(p.EnterTime)
		}

		var totalFloorTime time.Duration
		for _, d := range p.FloorClearTimes {
			totalFloorTime += d
		}
		avgFloorTime := time.Duration(0)
		if len(p.FloorClearTimes) > 0 {
			avgFloorTime = totalFloorTime / time.Duration(len(p.FloorClearTimes))
		}

		statStr := fmt.Sprintf("[%s, %s, %s]",
			formatDuration(timeSpent),
			formatDuration(avgFloorTime),
			formatDuration(p.BossKillTime),
		)

		if _, err := fmt.Fprintf(r.out, "[%s] %d %s HP:%d\n", p.State, p.ID, statStr, p.Health); err != nil {
			return err
		}
	}

	return nil
}

// formatIncomingEvent matches the domain event to its string representation.
//
//nolint:gocyclo // A single switch is the most idiomatic and readable way to map enum values
func formatIncomingEvent(e domain.Event) string {
	base := fmt.Sprintf("Player [%d]", e.PlayerID)

	switch e.ID {
	case domain.EventRegistered:
		return base + " registered"
	case domain.EventEntered:
		return base + " entered the dungeon"
	case domain.EventKilledMonster:
		return base + " killed the monster"
	case domain.EventNextFloor:
		return base + " went to the next floor"
	case domain.EventPrevFloor:
		return base + " went to the previous floor"
	case domain.EventEnteredBoss:
		return base + " entered the boss's floor"
	case domain.EventKilledBoss:
		return base + " killed the boss"
	case domain.EventLeftDungeon:
		return base + " left the dungeon"
	case domain.EventCannotContinue:
		return base + fmt.Sprintf(" cannot continue due to [%s]", e.ExtraParam)
	case domain.EventRestoredHealth:
		return base + fmt.Sprintf(" has restored [%s] of health", e.ExtraParam)
	case domain.EventReceivedDamage:
		return base + fmt.Sprintf(" recieved [%s] of damage", e.ExtraParam) // As it was in the task
	default:
		return base + " performed an unknown action"
	}
}

// formatDuration converts a time.Duration into the required [HH:MM:SS] string format.
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "00:00:00"
	}

	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
