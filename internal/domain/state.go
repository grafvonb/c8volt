package domain

import (
	"strings"
)

type State string

const (
	StateAll        State = "ALL"
	StateActive     State = "ACTIVE"     // pi, inc
	StateCompleted  State = "COMPLETED"  // pi
	StateCanceled   State = "CANCELED"   // pi
	StateTerminated State = "TERMINATED" // pi
	StateAbsent     State = "ABSENT"     // pi
	StateMigrated   State = "MIGRATED"   // inc
	StateResolved   State = "RESOLVED"   // inc
	StatePending    State = "PENDING"    // inc
)

func (s State) String() string { return string(s) }

func (s State) EqualsIgnoreCase(other State) bool {
	return strings.EqualFold(s.String(), other.String())
}

func (s State) In(states ...State) bool {
	for _, st := range states {
		if s.EqualsIgnoreCase(st) {
			return true
		}
	}
	return false
}

func (s State) IsTerminal() bool {
	return s.In(StateCompleted, StateCanceled, StateTerminated, StateAbsent)
}

type States []State

func (sx States) Contains(state State) bool {
	for _, s := range sx {
		if s.EqualsIgnoreCase(state) {
			return true
		}
	}
	return false
}

func (sx States) Strings() []string {
	out := make([]string, len(sx))
	for i, s := range sx {
		out[i] = s.String()
	}
	return out
}

func (sx States) String() string {
	return strings.Join(sx.Strings(), ", ")
}
