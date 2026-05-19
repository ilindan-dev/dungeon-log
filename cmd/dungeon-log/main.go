package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/ilindan-dev/dungeon-log/internal/adapters/config"
	"github.com/ilindan-dev/dungeon-log/internal/adapters/parser"
	"github.com/ilindan-dev/dungeon-log/internal/adapters/reporter"
	"github.com/ilindan-dev/dungeon-log/internal/service"
)

func main() {
	var configPath string
	var eventsPath string
	var outputPath string
	var reportPath string

	rootCmd := &cobra.Command{
		Use:          "dungeon-log",
		Short:        "Dungeon challenge log processor",
		Long:         `Parses dungeon events and generates a final report based on challenge rules.`,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := config.MustLoad(configPath)

			p, err := parser.NewFileParser(eventsPath)
			if err != nil {
				return fmt.Errorf("failed to initialize parser: %w", err)
			}
			defer func() { _ = p.Close() }()

			logOut := io.Writer(os.Stdout)
			reportOut := io.Writer(os.Stdout)

			if outputPath != "" {
				//nolint:gosec // This is a CLI tool, receiving file paths from the user is intended behavior
				fileLog, err := os.Create(outputPath)
				if err != nil {
					return fmt.Errorf("failed to open output log file: %w", err)
				}
				defer func() {
					_ = fileLog.Close()
				}()
				logOut = fileLog
			}

			if reportPath != "" {
				//nolint:gosec // This is a CLI tool, receiving file paths from the user is intended behavior
				fileReport, err := os.Create(reportPath)
				if err != nil {
					return fmt.Errorf("failed to open report file: %w", err)
				}
				defer func() {
					_ = fileReport.Close()
				}()
				reportOut = fileReport
			}

			r := reporter.NewReporter(logOut, reportOut)

			svc := service.NewDungeonService(cfg, p, r)
			if err := svc.Run(); err != nil {
				return fmt.Errorf("application error: %w", err)
			}

			return nil
		},
	}

	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file (required)")
	rootCmd.Flags().StringVarP(&eventsPath, "events", "e", "", "Path to events log file (required)")
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Path to output stream file (default: stdout)")
	rootCmd.Flags().StringVarP(&reportPath, "report", "r", "", "Path to final report file (default: stdout)")

	_ = rootCmd.MarkFlagRequired("config")
	_ = rootCmd.MarkFlagRequired("events")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
