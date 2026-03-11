package entityswitch

import (
	"encoding/json"
	"fmt"
	"sync"

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

func (State) CommandResponsePayloadKind() string { return Type }

type Command struct {
	Type string `json:"type"`
}

func (Command) CommandRequestPayloadKind() string { return Type }

type Event struct {
	Type             string   `json:"type"`
	AvailableActions []string `json:"available_actions,omitempty"`
	Cause            string   `json:"cause,omitempty"`
}

func init() {
	types.RegisterDomain(Describe())
	types.RegisterStateToCommands(Type, func(stateJSON json.RawMessage) ([]json.RawMessage, error) {
		var st State
		if err := json.Unmarshal(stateJSON, &st); err != nil {
			return nil, err
		}
		cmds := CommandsFromState(st)
		out := make([]json.RawMessage, 0, len(cmds))
		for _, c := range cmds {
			b, err := json.Marshal(c)
			if err != nil {
				return nil, err
			}
			out = append(out, b)
		}
		return out, nil
	})
}

// CommandsFromState returns the single Command needed to reproduce the given State.
func CommandsFromState(st State) []Command {
	if st.Power {
		return []Command{{Type: ActionTurnOn}}
	}
	return []Command{{Type: ActionTurnOff}}
}

func Describe() types.DomainDescriptor {
	actions := []types.ActionDescriptor{
		{Action: ActionTurnOn},
		{Action: ActionTurnOff},
	}
	return types.DomainDescriptor{
		Domain:   Type,
		Commands: actions,
		Events:   actions,
	}
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
	mu     *sync.RWMutex
}

func Bind(entity *types.Entity) Store {
	return Store{entity: entity, mu: &sync.RWMutex{}}
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
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := decodeState(s.entity.Data.Desired)
	if err != nil {
		return err
	}
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
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := decodeState(s.entity.Data.Reported)
	if err != nil {
		return err
	}
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

func (s Store) readDesired() (State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return decodeState(s.entity.Data.Desired)
}

func (s Store) readReported() (State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return decodeState(s.entity.Data.Reported)
}

func decodeState(raw json.RawMessage) (State, error) {
	if len(raw) == 0 {
		return State{}, nil
	}
	var st State
	return st, json.Unmarshal(raw, &st)
}
