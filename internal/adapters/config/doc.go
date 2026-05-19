// Package config implements the configuration adapter for the application.
//
// In this architecture, configuration from external sources (like JSON files)
// is treated as an input mechanism. This package is responsible for reading
// external files, parsing them, and mapping the raw data (DTOs) into the
// pure domain.Config entity required by the core business logic.
package config
