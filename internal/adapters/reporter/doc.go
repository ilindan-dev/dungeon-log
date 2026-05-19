// Package reporter provides the output formatting and delivery mechanisms.
//
// It implements the core ports.Reporter interface, translating pure domain
// events, player states, and challenge statistics into human-readable text
// streams. By abstracting the target destination (e.g., standard output or
// local files via io.Writer), it ensures that the core business logic remains
// entirely decoupled from how and where the execution results are presented.
package reporter
