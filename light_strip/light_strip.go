package light_strip

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/slidebolt/sdk-types"
)

const Type = "light_strip"

const (
	ActionTurnOn        = "turn_on"
	ActionTurnOff       = "turn_off"
	ActionSetBrightness = "set_brightness"
	ActionSetRGB        = "set_rgb"
	ActionSetEffect     = "set_effect"
	ActionSetSegment    = "set_segment"
	ActionClearSegments = "clear_segments"
)

const (
	ColorModeRGB     = "rgb"
	ColorModeEffect  = "effect"
	ColorModeSegment = "segment"
)

const (
	MaxBrightness   = 100
	MinBrightness   = 0
	MaxRGBLength    = 3
	MaxEffectLength = 256
)

// Segment represents a single addressable zone within the strip.
// Index is the zero-based position; RGB and Brightness are optional overrides.
type Segment struct {
	Index      int   `json:"index"`
	RGB        []int `json:"rgb,omitempty"`
	Brightness int   `json:"brightness,omitempty"`
}

type State struct {
	Power       bool      `json:"power"`
	Brightness  int       `json:"brightness,omitempty"`
	RGB         []int     `json:"rgb,omitempty"`
	Effect      string    `json:"effect,omitempty"`
	EffectSpeed int       `json:"effect_speed,omitempty"`
	ColorMode   string    `json:"color_mode,omitempty"`
	Segments    []Segment `json:"segments,omitempty"`
}

func (State) CommandResponsePayloadKind() string { return Type }

type Command struct {
	Type        string   `json:"type"`
	Brightness  *int     `json:"brightness,omitempty"`
	RGB         *[]int   `json:"rgb,omitempty"`
	Effect      *string  `json:"effect,omitempty"`
	EffectSpeed *int     `json:"effect_speed,omitempty"`
	Segment     *Segment `json:"segment,omitempty"`
}

func (Command) CommandRequestPayloadKind() string { return Type }

type Event struct {
	Type             string   `json:"type"`
	Brightness       *int     `json:"brightness,omitempty"`
	RGB              *[]int   `json:"rgb,omitempty"`
	Effect           *string  `json:"effect,omitempty"`
	EffectSpeed      *int     `json:"effect_speed,omitempty"`
	ColorMode        *string  `json:"color_mode,omitempty"`
	Segment          *Segment `json:"segment,omitempty"`
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
	Power       bool      `json:"power"`
	Brightness  *int      `json:"brightness,omitempty"`
	RGB         *[]int    `json:"rgb,omitempty"`
	Effect      *string   `json:"effect,omitempty"`
	EffectSpeed *int      `json:"effect_speed,omitempty"`
	ColorMode   *string   `json:"color_mode,omitempty"`
	Segments    []Segment `json:"segments,omitempty"`
}

func commandsFromStateJSON(stateJSON json.RawMessage) ([]Command, error) {
	var st stateWithPresence
	if err := json.Unmarshal(stateJSON, &st); err != nil {
		return nil, err
	}

	cmds := make([]Command, 0, 8)
	if st.Power {
		cmds = append(cmds, Command{Type: ActionTurnOn})
	} else {
		cmds = append(cmds, Command{Type: ActionTurnOff})
	}
	if st.Brightness != nil {
		b := *st.Brightness
		cmds = append(cmds, Command{Type: ActionSetBrightness, Brightness: &b})
	}

	mode := inferColorModeWithPresence(st.ColorMode, st.RGB, st.Effect, st.Segments)
	switch mode {
	case ColorModeRGB:
		if st.RGB != nil && len(*st.RGB) == 3 {
			rgb := cloneInts(*st.RGB)
			cmds = append(cmds, Command{Type: ActionSetRGB, RGB: &rgb})
		}
	case ColorModeEffect:
		if st.Effect != nil && *st.Effect != "" {
			eff := *st.Effect
			cmd := Command{Type: ActionSetEffect, Effect: &eff}
			if st.EffectSpeed != nil {
				speed := *st.EffectSpeed
				cmd.EffectSpeed = &speed
			}
			cmds = append(cmds, cmd)
		}
	case ColorModeSegment:
		for _, seg := range st.Segments {
			s := cloneSegment(seg)
			cmds = append(cmds, Command{Type: ActionSetSegment, Segment: &s})
		}
	}
	return cmds, nil
}

// CommandsFromState decomposes a State into the minimal ordered slice of
// Commands needed to reproduce it. Power is always first so hardware reaches
// the correct on/off state before attribute changes are applied.
func CommandsFromState(st State) []Command {
	cmds := make([]Command, 0, 8)
	if st.Power {
		cmds = append(cmds, Command{Type: ActionTurnOn})
	} else {
		cmds = append(cmds, Command{Type: ActionTurnOff})
	}
	if st.Brightness != 0 {
		b := st.Brightness
		cmds = append(cmds, Command{Type: ActionSetBrightness, Brightness: &b})
	}

	switch st.ColorMode {
	case ColorModeRGB:
		if len(st.RGB) == 3 {
			rgb := cloneInts(st.RGB)
			cmds = append(cmds, Command{Type: ActionSetRGB, RGB: &rgb})
		}
	case ColorModeEffect:
		if st.Effect != "" {
			eff := st.Effect
			cmd := Command{Type: ActionSetEffect, Effect: &eff}
			if st.EffectSpeed != 0 {
				speed := st.EffectSpeed
				cmd.EffectSpeed = &speed
			}
			cmds = append(cmds, cmd)
		}
	case ColorModeSegment:
		for _, seg := range st.Segments {
			s := cloneSegment(seg)
			cmds = append(cmds, Command{Type: ActionSetSegment, Segment: &s})
		}
	default:
		// Legacy fallback: no color_mode set, emit whatever is present.
		if len(st.RGB) == 3 {
			rgb := cloneInts(st.RGB)
			cmds = append(cmds, Command{Type: ActionSetRGB, RGB: &rgb})
		}
		if st.Effect != "" {
			eff := st.Effect
			cmd := Command{Type: ActionSetEffect, Effect: &eff}
			if st.EffectSpeed != 0 {
				speed := st.EffectSpeed
				cmd.EffectSpeed = &speed
			}
			cmds = append(cmds, cmd)
		}
		for _, seg := range st.Segments {
			s := cloneSegment(seg)
			cmds = append(cmds, Command{Type: ActionSetSegment, Segment: &s})
		}
	}
	return cmds
}

func Describe() types.DomainDescriptor {
	brightness := []types.FieldDescriptor{{Name: "brightness", Type: "int", Required: true, Min: intPtr(0), Max: intPtr(100)}}
	rgb := []types.FieldDescriptor{{Name: "rgb", Type: "[]int", Required: true}}
	effect := []types.FieldDescriptor{
		{Name: "effect", Type: "string", Required: true},
		{Name: "effect_speed", Type: "int", Required: false},
	}
	segment := []types.FieldDescriptor{
		{Name: "segment", Type: "object", Required: true},
	}

	actions := []types.ActionDescriptor{
		{Action: ActionTurnOn},
		{Action: ActionTurnOff},
		{Action: ActionSetBrightness, Fields: brightness},
		{Action: ActionSetRGB, Fields: rgb},
		{Action: ActionSetEffect, Fields: effect},
		{Action: ActionSetSegment, Fields: segment},
		{Action: ActionClearSegments},
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
		ActionSetEffect,
		ActionSetSegment,
		ActionClearSegments,
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
	case ActionTurnOn, ActionTurnOff, ActionClearSegments:
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
	case ActionSetEffect:
		if c.Effect == nil || *c.Effect == "" {
			return fmt.Errorf("effect required for %s", ActionSetEffect)
		}
		if len(*c.Effect) > MaxEffectLength {
			return fmt.Errorf("effect name too long (max %d)", MaxEffectLength)
		}
		return nil
	case ActionSetSegment:
		if c.Segment == nil {
			return fmt.Errorf("segment required for %s", ActionSetSegment)
		}
		if c.Segment.Index < 0 {
			return fmt.Errorf("segment index must be >= 0")
		}
		if len(c.Segment.RGB) > 0 {
			if len(c.Segment.RGB) != MaxRGBLength {
				return fmt.Errorf("segment rgb must have %d components", MaxRGBLength)
			}
			for _, v := range c.Segment.RGB {
				if v < 0 || v > 255 {
					return fmt.Errorf("segment rgb component %d out of bounds (0-255)", v)
				}
			}
		}
		if c.Segment.Brightness != 0 {
			if c.Segment.Brightness < MinBrightness || c.Segment.Brightness > MaxBrightness {
				return fmt.Errorf("segment brightness must be between %d and %d", MinBrightness, MaxBrightness)
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported light_strip command: %s", c.Type)
	}
}

func ValidateEvent(e Event) error {
	switch e.Type {
	case ActionTurnOn, ActionTurnOff, ActionClearSegments:
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
	case ActionSetEffect:
		if e.Effect == nil || *e.Effect == "" {
			return fmt.Errorf("effect required for %s", ActionSetEffect)
		}
		if len(*e.Effect) > MaxEffectLength {
			return fmt.Errorf("effect name too long (max %d)", MaxEffectLength)
		}
		return nil
	case ActionSetSegment:
		if e.Segment == nil {
			return fmt.Errorf("segment required for %s", ActionSetSegment)
		}
		if e.Segment.Index < 0 {
			return fmt.Errorf("segment index must be >= 0")
		}
		if len(e.Segment.RGB) > 0 {
			if len(e.Segment.RGB) != MaxRGBLength {
				return fmt.Errorf("segment rgb must have %d components", MaxRGBLength)
			}
			for _, v := range e.Segment.RGB {
				if v < 0 || v > 255 {
					return fmt.Errorf("segment rgb component %d out of bounds (0-255)", v)
				}
			}
		}
		if e.Segment.Brightness != 0 {
			if e.Segment.Brightness < MinBrightness || e.Segment.Brightness > MaxBrightness {
				return fmt.Errorf("segment brightness must be between %d and %d", MinBrightness, MaxBrightness)
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported light_strip event: %s", e.Type)
	}
}

// Store binds to an Entity and manages desired/reported/effective light_strip state.
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
		st.Segments = nil
	case ActionSetEffect:
		st.Effect = *cmd.Effect
		if cmd.EffectSpeed != nil {
			st.EffectSpeed = *cmd.EffectSpeed
		}
		st.ColorMode = ColorModeEffect
		st.Segments = nil
	case ActionSetSegment:
		upsertSegment(&st, *cmd.Segment)
		st.ColorMode = ColorModeSegment
	case ActionClearSegments:
		st.Segments = nil
		st.ColorMode = ""
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
	case ActionClearSegments:
		st.Segments = nil
		st.ColorMode = ""
	}
	// Preserve optional attribute updates regardless of event type.
	if evt.Brightness != nil {
		st.Brightness = *evt.Brightness
	}
	if evt.RGB != nil {
		st.RGB = cloneInts(*evt.RGB)
	}
	if evt.Effect != nil {
		st.Effect = *evt.Effect
	}
	if evt.EffectSpeed != nil {
		st.EffectSpeed = *evt.EffectSpeed
	}
	if evt.Segment != nil {
		upsertSegment(&st, *evt.Segment)
	}
	if evt.ColorMode != nil {
		st.ColorMode = *evt.ColorMode
	} else {
		// Infer active mode when providers send mixed events with attributes.
		if evt.Segment != nil {
			st.ColorMode = ColorModeSegment
		} else if evt.Effect != nil && *evt.Effect != "" {
			st.ColorMode = ColorModeEffect
		} else if evt.RGB != nil && len(*evt.RGB) == 3 {
			st.ColorMode = ColorModeRGB
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
func (s Store) SetEffect(name string, speed *int) error {
	return s.SetDesiredFromCommand(Command{Type: ActionSetEffect, Effect: &name, EffectSpeed: speed})
}
func (s Store) SetSegment(seg Segment) error {
	return s.SetDesiredFromCommand(Command{Type: ActionSetSegment, Segment: &seg})
}
func (s Store) ClearSegments() error {
	return s.SetDesiredFromCommand(Command{Type: ActionClearSegments})
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

func cloneSegment(seg Segment) Segment {
	out := Segment{Index: seg.Index, Brightness: seg.Brightness}
	if len(seg.RGB) > 0 {
		out.RGB = cloneInts(seg.RGB)
	}
	return out
}

// upsertSegment finds an existing segment by index and updates it, or appends a new one.
func upsertSegment(st *State, seg Segment) {
	for i, s := range st.Segments {
		if s.Index == seg.Index {
			st.Segments[i] = cloneSegment(seg)
			return
		}
	}
	st.Segments = append(st.Segments, cloneSegment(seg))
}

func inferColorModeWithPresence(explicit *string, rgb *[]int, effect *string, segments []Segment) string {
	if explicit != nil && *explicit != "" {
		return *explicit
	}
	if len(segments) > 0 {
		return ColorModeSegment
	}
	if effect != nil && *effect != "" {
		return ColorModeEffect
	}
	if rgb != nil && len(*rgb) == 3 {
		return ColorModeRGB
	}
	return ""
}
