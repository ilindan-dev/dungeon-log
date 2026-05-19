package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	"github.com/ilindan-dev/dungeon-log/internal/core/domain"
)

// configDTO is the Data Transfer Object used to parse the raw JSON file.
// It remains unexported to ensure external layers only interact with domain.Config.
type configDTO struct {
	Floors   int    `json:"Floors" env-required:"true"`
	Monsters int    `json:"Monsters" env-required:"true"`
	OpenAt   string `json:"OpenAt" env-required:"true"`
	Duration int    `json:"Duration" env-required:"true"`
}

// MustLoad reads the configuration from the specified file path,
// parses it, and maps it into a pure domain.Config entity.
// It will fatally terminate the application if the file cannot be read
// or if the time formats are invalid, as the application cannot run without rules.
func MustLoad(path string) *domain.Config {
	var dto configDTO

	if err := cleanenv.ReadConfig(path, &dto); err != nil {
		log.Fatalf("cannot read config file: %v", err)
	}

	openAt, err := time.Parse("15:04:05", dto.OpenAt)
	if err != nil {
		log.Fatalf("invalid OpenAt time format (expected HH:MM:SS): %v", err)
	}

	return &domain.Config{
		Floors:   dto.Floors,
		Monsters: dto.Monsters,
		OpenAt:   openAt,
		Duration: time.Duration(dto.Duration) * time.Hour,
	}
}
