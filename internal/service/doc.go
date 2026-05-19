// Package service implements the core business logic and state orchestration
// for the dungeon challenge.
//
// The core mechanism is a manually implemented, switch-driven finite state
// machine (FSM) that chronologically processes domain.Event streams, mutates
// player states, enforces domain rules (e.g., preventing impossible moves),
// and triggers system-generated outgoing events via the reporter port.
package service
