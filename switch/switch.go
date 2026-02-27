package entityswitch

import (
	"encoding/json"
	"fmt"

	"github.com/slidebolt/sdk-types"
)

const Type = "switch"

const (
	ActionTurnOn  = "turn_on"
	ActionTurnOff = "turn_off"
)

type State struct {
	Power bool `json:"power"`
}

type Command struct {
	Type string `json:"type"`
}

type Event struct {
	Type             string   `json:"type"`
	AvailableActions []string `json:"available_actions,omitempty"`
	Cause            string   `json:"cause,omitempty"`
}

func SupportedActions() []string {
	return []string{ActionTurnOn, ActionTurnOff}
}

func ParseCommand(cmd types.Command) (Command, error) {
	var c Command
	if err := json.Unmarshal(cmd.Payload, &c); err != nil {
		return c, err
	}
	return c, ValidateCommand(c)
}

func ParseEvent(evt types.Event) (Event, error) {
	var e Event
	if err := json.Unmarshal(evt.Payload, &e); err != nil {
		return e, err
	}
	return e, ValidateEvent(e)
}

func ValidateCommand(c Command) error {
	switch c.Type {
	case ActionTurnOn, ActionTurnOff:
		return nil
	default:
		return fmt.Errorf("unsupported switch command: %s", c.Type)
	}
}

func ValidateEvent(e Event) error {
	switch e.Type {
	case ActionTurnOn, ActionTurnOff:
		return nil
	default:
		return fmt.Errorf("unsupported switch event: %s", e.Type)
	}
}

// Store binds to an Entity and manages desired/reported switch state.
type Store struct {
	entity *types.Entity
}

func Bind(entity *types.Entity) Store {
	return Store{entity: entity}
}

func (s Store) EnsureDefaultActions() {
	if len(s.entity.Actions) == 0 {
		s.entity.Actions = SupportedActions()
	}
}

func (s Store) Supports(action string) bool {
	for _, a := range s.entity.Actions {
		if a == action {
			return true
		}
	}
	return false
}

func (s Store) SetDesiredFromCommand(cmd Command) error {
	st, _ := s.readDesired()
	switch cmd.Type {
	case ActionTurnOn:
		st.Power = true
	case ActionTurnOff:
		st.Power = false
	}
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	s.entity.Data.Desired = b
	return nil
}

func (s Store) SetReportedFromEvent(evt Event) error {
	st, _ := s.readReported()
	switch evt.Type {
	case ActionTurnOn:
		st.Power = true
	case ActionTurnOff:
		st.Power = false
	}
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	s.entity.Data.Reported = b
	s.entity.Data.Effective = b
	return nil
}

func (s Store) readDesired() (State, error)  { return decodeState(s.entity.Data.Desired) }
func (s Store) readReported() (State, error) { return decodeState(s.entity.Data.Reported) }

func decodeState(raw json.RawMessage) (State, error) {
	if len(raw) == 0 {
		return State{}, nil
	}
	var st State
	return st, json.Unmarshal(raw, &st)
}
