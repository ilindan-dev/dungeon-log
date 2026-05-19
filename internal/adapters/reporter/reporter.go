package reporter

import (
	"fmt"
	"io"
	"time"

	"github.com/ilindan-dev/dungeon-log/internal/core/domain"
)

// BaseReporter implements the ports.Reporter interface.
// It routes event logs and final business reports to their respective writers.
type BaseReporter struct {
	logOut    io.Writer
	reportOut io.Writer
}

// NewReporter creates a new reporter with specified writers for logs and reports.
func NewReporter(logOut, reportOut io.Writer) *BaseReporter {
	return &BaseReporter{
		logOut:    logOut,
		reportOut: reportOut,
	}
}

// EmitIncoming formats and prints a valid action taken by a player.
func (r *BaseReporter) EmitIncoming(e domain.Event) error {
	_, err := fmt.Fprintf(r.logOut, "[%s] %s\n", e.Time.Format(time.TimeOnly), formatIncomingEvent(e))
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

	_, err := fmt.Fprintf(r.logOut, "[%s] %s\n", timeStr, msg)
	return err
}

// PrintFinalReport calculates statistics and outputs the final summary for all players.
func (r *BaseReporter) PrintFinalReport(players []*domain.Player) error {
	if _, err := fmt.Fprintln(r.reportOut, "Final report:"); err != nil {
		return err
	}

	for _, p := range players {
		statStr := fmt.Sprintf("[%s, %s, %s]",
			formatDuration(p.TimeSpent),
			formatDuration(p.AvgFloorClearTime),
			formatDuration(p.BossKillTime),
		)

		if _, err := fmt.Fprintf(r.reportOut, "[%s] %d %s HP:%d\n", p.State, p.ID, statStr, p.Health); err != nil {
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
