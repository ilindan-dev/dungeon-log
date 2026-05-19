package parser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ilindan-dev/dungeon-log/internal/core/domain"
)

// logPattern matches lines like: [14:27:00] 2 11 60
// Group 1: Time, Group 2: PlayerID, Group 3: EventID, Group 4: ExtraParam (optional)
var logPattern = regexp.MustCompile(`^\[(\d{2}:\d{2}:\d{2})\]\s+(\d+)\s+(\d+)(?:\s+(.*))?$`)

// FileParser implements ports.EventParser by reading events from a text file.
type FileParser struct {
	file    *os.File
	scanner *bufio.Scanner
}

// NewFileParser opens the file and initializes the scanner.
func NewFileParser(path string) (*FileParser, error) {
	//nolint:gosec // This is a CLI tool, receiving file paths from the user is intended behavior
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open events file: %w", err)
	}

	return &FileParser{
		file:    f,
		scanner: bufio.NewScanner(f),
	}, nil
}

// Close ensures the underlying file descriptor is released.
func (p *FileParser) Close() error {
	return p.file.Close()
}

// Next reads the next line and parses it into a domain.Event.
func (p *FileParser) Next() (domain.Event, error) {
	if !p.scanner.Scan() {
		err := p.scanner.Err()
		if err == nil {
			return domain.Event{}, io.EOF
		}
		return domain.Event{}, fmt.Errorf("scanner error: %w", err)
	}

	line := strings.TrimSpace(p.scanner.Text())
	if line == "" {
		return p.Next() // Skip empty lines
	}

	return p.parseLine(line)
}

func (p *FileParser) parseLine(line string) (domain.Event, error) {
	matches := logPattern.FindStringSubmatch(line)
	if matches == nil {
		return domain.Event{}, fmt.Errorf("invalid log format: %s", line)
	}

	parsedTime, err := time.Parse("15:04:05", matches[1])
	if err != nil {
		return domain.Event{}, fmt.Errorf("invalid time '%s': %w", matches[1], err)
	}

	playerID, err := strconv.Atoi(matches[2])
	if err != nil {
		return domain.Event{}, fmt.Errorf("invalid PlayerID '%s': %w", matches[2], err)
	}

	eventIDRaw, err := strconv.Atoi(matches[3])
	if err != nil {
		return domain.Event{}, fmt.Errorf("invalid EventID '%s': %w", matches[3], err)
	}

	return domain.Event{
		ID:         domain.EventID(eventIDRaw),
		PlayerID:   playerID,
		ExtraParam: strings.TrimSpace(matches[4]),
		Time:       parsedTime,
	}, nil
}
