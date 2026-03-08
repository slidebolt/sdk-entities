package light

import (
	"encoding/json"
	"fmt"

	"github.com/slidebolt/sdk-types"
)

const Type = "light"

const (
	ActionTurnOn         = "turn_on"
	ActionTurnOff        = "turn_off"
	ActionSetBrightness  = "set_brightness"
	ActionSetRGB         = "set_rgb"
	ActionSetTemperature = "set_temperature"
	ActionSetScene       = "set_scene"
)

type State struct {
	Power       bool   `json:"power"`
	Brightness  int    `json:"brightness,omitempty"`
	RGB         []int  `json:"rgb,omitempty"`
	Temperature int    `json:"temperature,omitempty"`
	Scene       string `json:"scene,omitempty"`
}

func (State) CommandResponsePayloadKind() string { return Type }

type Command struct {
	Type        string  `json:"type"`
	Brightness  *int    `json:"brightness,omitempty"`
	RGB         *[]int  `json:"rgb,omitempty"`
	Temperature *int    `json:"temperature,omitempty"`
	Scene       *string `json:"scene,omitempty"`
}

func (Command) CommandRequestPayloadKind() string { return Type }

type Event struct {
	Type             string   `json:"type"`
	Brightness       *int     `json:"brightness,omitempty"`
	RGB              *[]int   `json:"rgb,omitempty"`
	Temperature      *int     `json:"temperature,omitempty"`
	Scene            *string  `json:"scene,omitempty"`
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

// CommandsFromState decomposes a State into the minimal ordered slice of
// Commands needed to reproduce it. Power is always first so hardware reaches
// the correct on/off state before attribute changes are applied.
func CommandsFromState(st State) []Command {
	cmds := make([]Command, 0, 5)
	if st.Power {
		cmds = append(cmds, Command{Type: ActionTurnOn})
	} else {
		cmds = append(cmds, Command{Type: ActionTurnOff})
	}
	if st.Brightness != 0 {
		b := st.Brightness
		cmds = append(cmds, Command{Type: ActionSetBrightness, Brightness: &b})
	}
	if len(st.RGB) == 3 {
		rgb := cloneInts(st.RGB)
		cmds = append(cmds, Command{Type: ActionSetRGB, RGB: &rgb})
	}
	if st.Temperature != 0 {
		temp := st.Temperature
		cmds = append(cmds, Command{Type: ActionSetTemperature, Temperature: &temp})
	}
	if st.Scene != "" {
		scene := st.Scene
		cmds = append(cmds, Command{Type: ActionSetScene, Scene: &scene})
	}
	return cmds
}

func Describe() types.DomainDescriptor {
	brightness := []types.FieldDescriptor{{Name: "brightness", Type: "int", Required: true, Min: intPtr(0), Max: intPtr(100)}}
	rgb := []types.FieldDescriptor{{Name: "rgb", Type: "[]int", Required: true}}
	temperature := []types.FieldDescriptor{{Name: "temperature", Type: "int", Required: true}}
	scene := []types.FieldDescriptor{{Name: "scene", Type: "string", Required: true}}

	actions := []types.ActionDescriptor{
		{Action: ActionTurnOn},
		{Action: ActionTurnOff},
		{Action: ActionSetBrightness, Fields: brightness},
		{Action: ActionSetRGB, Fields: rgb},
		{Action: ActionSetTemperature, Fields: temperature},
		{Action: ActionSetScene, Fields: scene},
	}

	return types.DomainDescriptor{
		Domain:   Type,
		Commands: actions,
		Events:   actions,
	}
}

func intPtr(v int) *int { return &v }

func SupportedActions() []string {
	return []string{
		ActionTurnOn,
		ActionTurnOff,
		ActionSetBrightness,
		ActionSetRGB,
		ActionSetTemperature,
		ActionSetScene,
	}
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
	case ActionSetBrightness:
		if c.Brightness == nil {
			return fmt.Errorf("brightness required for %s", ActionSetBrightness)
		}
		if *c.Brightness < 0 || *c.Brightness > 100 {
			return fmt.Errorf("brightness must be between 0 and 100")
		}
		return nil
	case ActionSetRGB:
		if c.RGB == nil || len(*c.RGB) != 3 {
			return fmt.Errorf("rgb[3] required for %s", ActionSetRGB)
		}
		return nil
	case ActionSetTemperature:
		if c.Temperature == nil {
			return fmt.Errorf("temperature required for %s", ActionSetTemperature)
		}
		return nil
	case ActionSetScene:
		if c.Scene == nil || *c.Scene == "" {
			return fmt.Errorf("scene required for %s", ActionSetScene)
		}
		return nil
	default:
		return fmt.Errorf("unsupported light command: %s", c.Type)
	}
}

func ValidateEvent(e Event) error {
	switch e.Type {
	case ActionTurnOn, ActionTurnOff:
		return nil
	case ActionSetBrightness:
		if e.Brightness == nil {
			return fmt.Errorf("brightness required for %s", ActionSetBrightness)
		}
		if *e.Brightness < 0 || *e.Brightness > 100 {
			return fmt.Errorf("brightness must be between 0 and 100")
		}
		return nil
	case ActionSetRGB:
		if e.RGB == nil || len(*e.RGB) != 3 {
			return fmt.Errorf("rgb[3] required for %s", ActionSetRGB)
		}
		return nil
	case ActionSetTemperature:
		if e.Temperature == nil {
			return fmt.Errorf("temperature required for %s", ActionSetTemperature)
		}
		return nil
	case ActionSetScene:
		if e.Scene == nil || *e.Scene == "" {
			return fmt.Errorf("scene required for %s", ActionSetScene)
		}
		return nil
	default:
		return fmt.Errorf("unsupported light event: %s", e.Type)
	}
}

// Store binds to an Entity and manages desired/reported/effective light state.
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

func (s Store) Desired() (State, error)  { return decodeState(s.entity.Data.Desired) }
func (s Store) Reported() (State, error) { return decodeState(s.entity.Data.Reported) }

func (s Store) SetDesiredFromCommand(cmd Command) error {
	st, err := s.Desired()
	if err != nil {
		return err
	}
	switch cmd.Type {
	case ActionTurnOn:
		st.Power = true
	case ActionTurnOff:
		st.Power = false
	case ActionSetBrightness:
		st.Brightness = *cmd.Brightness
	case ActionSetRGB:
		st.RGB = cloneInts(*cmd.RGB)
	case ActionSetTemperature:
		st.Temperature = *cmd.Temperature
	case ActionSetScene:
		st.Scene = *cmd.Scene
	}
	return s.writeDesired(st)
}

func (s Store) SetReportedFromEvent(evt Event) error {
	st, err := s.Reported()
	if err != nil {
		return err
	}
	switch evt.Type {
	case ActionTurnOn:
		st.Power = true
	case ActionTurnOff:
		st.Power = false
	case ActionSetBrightness:
		if evt.Brightness != nil {
			st.Brightness = *evt.Brightness
		}
	case ActionSetRGB:
		if evt.RGB != nil {
			st.RGB = cloneInts(*evt.RGB)
		}
	case ActionSetTemperature:
		if evt.Temperature != nil {
			st.Temperature = *evt.Temperature
		}
	case ActionSetScene:
		if evt.Scene != nil {
			st.Scene = *evt.Scene
		}
	}
	if err := s.writeReported(st); err != nil {
		return err
	}
	return s.writeEffective(st)
}

func (s Store) TurnOn() error  { return s.SetDesiredFromCommand(Command{Type: ActionTurnOn}) }
func (s Store) TurnOff() error { return s.SetDesiredFromCommand(Command{Type: ActionTurnOff}) }
func (s Store) SetBrightness(v int) error {
	return s.SetDesiredFromCommand(Command{Type: ActionSetBrightness, Brightness: &v})
}
func (s Store) SetRGB(r, g, b int) error {
	rgb := []int{r, g, b}
	return s.SetDesiredFromCommand(Command{Type: ActionSetRGB, RGB: &rgb})
}
func (s Store) SetTemperature(v int) error {
	return s.SetDesiredFromCommand(Command{Type: ActionSetTemperature, Temperature: &v})
}
func (s Store) SetScene(scene string) error {
	return s.SetDesiredFromCommand(Command{Type: ActionSetScene, Scene: &scene})
}

func decodeState(raw json.RawMessage) (State, error) {
	if len(raw) == 0 {
		return State{}, nil
	}
	var st State
	return st, json.Unmarshal(raw, &st)
}

func (s Store) writeDesired(st State) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	s.entity.Data.Desired = b
	return nil
}

func (s Store) writeReported(st State) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	s.entity.Data.Reported = b
	return nil
}

func (s Store) writeEffective(st State) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	s.entity.Data.Effective = b
	return nil
}

func cloneInts(src []int) []int {
	dst := make([]int, len(src))
	copy(dst, src)
	return dst
}
