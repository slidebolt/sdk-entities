package light

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

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

const (
	ColorModeRGB         = "rgb"
	ColorModeTemperature = "temperature"
	ColorModeScene       = "scene"
)

const (
	MaxRGBLength    = 3
	MaxSceneLength  = 256
	MaxBrightness   = 100
	MinBrightness   = 0
)

type State struct {
	Power       bool   `json:"power"`
	Brightness  int    `json:"brightness,omitempty"`
	RGB         []int  `json:"rgb,omitempty"`
	Temperature int    `json:"temperature,omitempty"`
	Scene       string `json:"scene,omitempty"`
	ColorMode   string `json:"color_mode,omitempty"`
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
	ColorMode        *string  `json:"color_mode,omitempty"`
	AvailableActions []string `json:"available_actions,omitempty"`
	Cause            string   `json:"cause,omitempty"`
}

func init() {
	types.RegisterDomain(Describe())
	types.RegisterStateToCommands(Type, func(stateJSON json.RawMessage) ([]json.RawMessage, error) {
		cmds, err := commandsFromStateJSON(stateJSON)
		if err != nil {
			return nil, err
		}
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

type stateWithPresence struct {
	Power       bool    `json:"power"`
	Brightness  *int    `json:"brightness,omitempty"`
	RGB         *[]int  `json:"rgb,omitempty"`
	Temperature *int    `json:"temperature,omitempty"`
	Scene       *string `json:"scene,omitempty"`
	ColorMode   *string `json:"color_mode,omitempty"`
}

func commandsFromStateJSON(stateJSON json.RawMessage) ([]Command, error) {
	var st stateWithPresence
	if err := json.Unmarshal(stateJSON, &st); err != nil {
		return nil, err
	}

	cmds := make([]Command, 0, 5)
	if st.Power {
		cmds = append(cmds, Command{Type: ActionTurnOn})
	} else {
		cmds = append(cmds, Command{Type: ActionTurnOff})
	}
	if st.Brightness != nil {
		b := *st.Brightness
		cmds = append(cmds, Command{Type: ActionSetBrightness, Brightness: &b})
	}
	mode := inferColorModeWithPresence(st.ColorMode, st.Scene, st.RGB, st.Temperature)
	switch mode {
	case ColorModeScene:
		if st.Scene != nil && *st.Scene != "" {
			scene := *st.Scene
			cmds = append(cmds, Command{Type: ActionSetScene, Scene: &scene})
		}
	case ColorModeRGB:
		if st.RGB != nil && len(*st.RGB) == 3 {
			rgb := cloneInts(*st.RGB)
			cmds = append(cmds, Command{Type: ActionSetRGB, RGB: &rgb})
		}
	case ColorModeTemperature:
		if st.Temperature != nil {
			temp := *st.Temperature
			cmds = append(cmds, Command{Type: ActionSetTemperature, Temperature: &temp})
		}
	}
	return cmds, nil
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
	mode, _ := normalizeColorMode(st.ColorMode)
	if mode == "" {
		// Legacy behavior for call sites that still rely on this helper directly.
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
	switch mode {
	case ColorModeScene:
		if st.Scene != "" {
			scene := st.Scene
			cmds = append(cmds, Command{Type: ActionSetScene, Scene: &scene})
		}
	case ColorModeRGB:
		if len(st.RGB) == 3 {
			rgb := cloneInts(st.RGB)
			cmds = append(cmds, Command{Type: ActionSetRGB, RGB: &rgb})
		}
	case ColorModeTemperature:
		if st.Temperature != 0 {
			temp := st.Temperature
			cmds = append(cmds, Command{Type: ActionSetTemperature, Temperature: &temp})
		}
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
		if *c.Brightness < MinBrightness || *c.Brightness > MaxBrightness {
			return fmt.Errorf("brightness must be between %d and %d", MinBrightness, MaxBrightness)
		}
		return nil
	case ActionSetRGB:
		if c.RGB == nil || len(*c.RGB) != MaxRGBLength {
			return fmt.Errorf("rgb[%d] required for %s", MaxRGBLength, ActionSetRGB)
		}
		for _, v := range *c.RGB {
			if v < 0 || v > 255 {
				return fmt.Errorf("rgb component %d out of bounds (0-255)", v)
			}
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
		if len(*c.Scene) > MaxSceneLength {
			return fmt.Errorf("scene name too long (max %d)", MaxSceneLength)
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
		if *e.Brightness < MinBrightness || *e.Brightness > MaxBrightness {
			return fmt.Errorf("brightness must be between %d and %d", MinBrightness, MaxBrightness)
		}
		return nil
	case ActionSetRGB:
		if e.RGB == nil || len(*e.RGB) != MaxRGBLength {
			return fmt.Errorf("rgb[%d] required for %s", MaxRGBLength, ActionSetRGB)
		}
		for _, v := range *e.RGB {
			if v < 0 || v > 255 {
				return fmt.Errorf("rgb component %d out of bounds (0-255)", v)
			}
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
		if len(*e.Scene) > MaxSceneLength {
			return fmt.Errorf("scene name too long (max %d)", MaxSceneLength)
		}
		return nil
	default:
		return fmt.Errorf("unsupported light event: %s", e.Type)
	}
}

// Store binds to an Entity and manages desired/reported/effective light state.
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

func (s Store) Desired() (State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return decodeState(s.entity.Data.Desired)
}

func (s Store) Reported() (State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return decodeState(s.entity.Data.Reported)
}

func (s Store) SetDesiredFromCommand(cmd Command) error {
	if err := ValidateCommand(cmd); err != nil {
		return err
	}

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
	case ActionSetBrightness:
		st.Brightness = *cmd.Brightness
	case ActionSetRGB:
		st.RGB = cloneInts(*cmd.RGB)
		st.ColorMode = ColorModeRGB
	case ActionSetTemperature:
		st.Temperature = *cmd.Temperature
		st.ColorMode = ColorModeTemperature
	case ActionSetScene:
		st.Scene = *cmd.Scene
		st.ColorMode = ColorModeScene
	}
	return s.writeDesired(st)
}

func (s Store) SetReportedFromEvent(evt Event) error {
	if err := ValidateEvent(evt); err != nil {
		return err
	}

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
	case ActionSetBrightness, ActionSetRGB, ActionSetTemperature, ActionSetScene:
	}
	// Preserve optional attribute updates regardless of event type. Some
	// providers attach brightness/rgb fields to turn_on events.
	if evt.Brightness != nil {
		st.Brightness = *evt.Brightness
	}
	if evt.RGB != nil {
		st.RGB = cloneInts(*evt.RGB)
	}
	if evt.Temperature != nil {
		st.Temperature = *evt.Temperature
	}
	if evt.Scene != nil {
		st.Scene = *evt.Scene
	}
	if evt.ColorMode != nil {
		if mode, ok := normalizeColorMode(*evt.ColorMode); ok {
			st.ColorMode = mode
		}
	} else {
		// Infer active mode when providers send mixed turn_on + attributes.
		if evt.Scene != nil && *evt.Scene != "" {
			st.ColorMode = ColorModeScene
		} else if evt.RGB != nil && len(*evt.RGB) == 3 {
			st.ColorMode = ColorModeRGB
		} else if evt.Temperature != nil {
			st.ColorMode = ColorModeTemperature
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

func normalizeColorMode(raw string) (string, bool) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch mode {
	case ColorModeRGB, "rgbw", "rgbww", "rgbcw":
		return ColorModeRGB, true
	case ColorModeTemperature, "color_temperature", "color_temp", "ct":
		return ColorModeTemperature, true
	case ColorModeScene, "effect":
		return ColorModeScene, true
	default:
		return "", false
	}
}

func inferColorModeWithPresence(explicit *string, scene *string, rgb *[]int, temperature *int) string {
	if explicit != nil {
		if mode, ok := normalizeColorMode(*explicit); ok {
			return mode
		}
	}
	if scene != nil && *scene != "" {
		return ColorModeScene
	}
	if rgb != nil && len(*rgb) == 3 {
		return ColorModeRGB
	}
	if temperature != nil {
		return ColorModeTemperature
	}
	return ""
}
