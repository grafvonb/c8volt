package process

import (
	"fmt"
	"strings"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
)

type State string

const (
	StateAll        State = "ALL"
	StateActive     State = "ACTIVE"
	StateCompleted  State = "COMPLETED"
	StateCanceled   State = "CANCELED"
	StateTerminated State = "TERMINATED"
	StateAbsent     State = "ABSENT"
	StateUnknown    State = "UNKNOWN"
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

func ParseState(in string) (State, bool) {
	switch strings.ToLower(in) {
	case "all":
		return StateAll, true
	case "active":
		return StateActive, true
	case "completed":
		return StateCompleted, true
	case "canceled", "cancelled":
		return StateCanceled, true
	case "terminated":
		return StateTerminated, true
	case "absent":
		return StateAbsent, true
	default:
		return StateUnknown, false
	}
}

func (s State) IsTerminal() bool {
	return s.In(StateCompleted, StateCanceled, StateTerminated)
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
	return strings.Join(sx.Strings(), ",")
}

func ParseStates(in []string) (States, error) {
	var out States
	for _, s := range in {
		parsed, ok := ParseState(s)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ferr.ErrInvalidState, s)
		}
		out = append(out, parsed)
	}
	return out, nil
}
