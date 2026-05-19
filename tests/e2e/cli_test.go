package e2e_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupBinary(t *testing.T) string {
	t.Helper()
	binaryName := "dungeon-log-e2e"
	buildCmd := exec.Command("go", "build", "-o", binaryName, "../../cmd/dungeon-log/main.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	return binaryName
}

func TestCLI_StandardFlow(t *testing.T) {
	binaryName := setupBinary(t)
	defer func() { _ = os.Remove(binaryName) }()

	configPath := filepath.Join("testdata", "config.json")
	eventsPath := filepath.Join("testdata", "events")
	expectedPath := filepath.Join("testdata", "expected_full")

	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read expected_full: %v", err)
	}
	expected := strings.TrimSpace(string(expectedBytes))

	runCmd := exec.Command("./"+binaryName, "-c", configPath, "-e", eventsPath)
	var outBuffer, errBuffer bytes.Buffer
	runCmd.Stdout = &outBuffer
	runCmd.Stderr = &errBuffer

	if err := runCmd.Run(); err != nil {
		t.Fatalf("Binary execution failed: %v\nStderr: %s", err, errBuffer.String())
	}

	got := strings.TrimSpace(outBuffer.String())
	if got != expected {
		t.Errorf("E2E test failed for stdout.\nEXPECTED:\n%s\n\nGOT:\n%s", expected, got)
	}
}

func TestCLI_SplitFlow(t *testing.T) {
	binaryName := setupBinary(t)
	defer func() { _ = os.Remove(binaryName) }()

	configPath := filepath.Join("testdata", "config.json")
	eventsPath := filepath.Join("testdata", "events")

	logOutPath := "temp_logs"
	reportOutPath := "temp_report"
	defer func() { _ = os.Remove(logOutPath) }()
	defer func() { _ = os.Remove(reportOutPath) }()

	runCmd := exec.Command("./"+binaryName, "-c", configPath, "-e", eventsPath, "-o", logOutPath, "-r", reportOutPath)
	if err := runCmd.Run(); err != nil {
		t.Fatalf("Binary execution failed: %v", err)
	}

	gotLogsBytes, _ := os.ReadFile(logOutPath)
	gotReportBytes, _ := os.ReadFile(reportOutPath)
	gotLogs := strings.TrimSpace(string(gotLogsBytes))
	gotReport := strings.TrimSpace(string(gotReportBytes))

	expectedLogsBytes, err := os.ReadFile(filepath.Join("testdata", "expected_logs"))
	if err != nil {
		t.Fatalf("Failed to read expected_logs: %v", err)
	}
	expectedReportBytes, err := os.ReadFile(filepath.Join("testdata", "expected_report"))
	if err != nil {
		t.Fatalf("Failed to read expected_report: %v", err)
	}

	expectedLogs := strings.TrimSpace(string(expectedLogsBytes))
	expectedReport := strings.TrimSpace(string(expectedReportBytes))

	if gotLogs != expectedLogs {
		t.Errorf("Logs output mismatch.\nEXPECTED:\n%s\n\nGOT:\n%s", expectedLogs, gotLogs)
	}

	if gotReport != expectedReport {
		t.Errorf("Report output mismatch.\nEXPECTED:\n%s\n\nGOT:\n%s", expectedReport, gotReport)
	}
}

func TestCLI_MissingFlags(t *testing.T) {
	binaryName := setupBinary(t)
	defer func() { _ = os.Remove(binaryName) }()

	runCmd := exec.Command("./" + binaryName)
	var errBuffer bytes.Buffer
	runCmd.Stderr = &errBuffer

	err := runCmd.Run()
	if err == nil {
		t.Fatalf("Expected app to fail without flags, but it succeeded")
	}

	if !strings.Contains(errBuffer.String(), `required flag(s) "config", "events" not set`) {
		t.Errorf("Expected Cobra error about missing flags, got: %s", errBuffer.String())
	}
}
