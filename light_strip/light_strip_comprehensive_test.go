package light_strip

import (
	"encoding/json"
	"testing"

	types "github.com/slidebolt/sdk-types"
)

func newEntity() *types.Entity {
	return &types.Entity{}
}

func TestStore_TurnOn(t *testing.T) {
	store := Bind(newEntity())
	if err := store.TurnOn(); err != nil {
		t.Fatalf("TurnOn failed: %v", err)
	}
	st, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}
	if !st.Power {
		t.Fatal("expected Power=true after TurnOn")
	}
}

func TestStore_TurnOff(t *testing.T) {
	entity := newEntity()
	store := Bind(entity)
	if err := store.TurnOn(); err != nil {
		t.Fatalf("TurnOn: %v", err)
	}
	if err := store.TurnOff(); err != nil {
		t.Fatalf("TurnOff: %v", err)
	}
	st, _ := store.Desired()
	if st.Power {
		t.Fatal("expected Power=false after TurnOff")
	}
}

func TestStore_SetBrightness(t *testing.T) {
	store := Bind(newEntity())
	if err := store.SetBrightness(80); err != nil {
		t.Fatalf("SetBrightness: %v", err)
	}
	st, _ := store.Desired()
	if st.Brightness != 80 {
		t.Fatalf("expected brightness=80, got %d", st.Brightness)
	}
}

func TestStore_SetRGB_SetsColorModeAndClearsSegments(t *testing.T) {
	entity := newEntity()
	store := Bind(entity)
	// First set some segments
	if err := store.SetSegment(Segment{Index: 0, RGB: []int{255, 0, 0}}); err != nil {
		t.Fatalf("SetSegment: %v", err)
	}
	// Now set RGB — should clear segments and set color mode
	if err := store.SetRGB(0, 255, 0); err != nil {
		t.Fatalf("SetRGB: %v", err)
	}
	st, _ := store.Desired()
	if st.ColorMode != ColorModeRGB {
		t.Fatalf("expected color_mode=%q, got %q", ColorModeRGB, st.ColorMode)
	}
	if len(st.Segments) != 0 {
		t.Fatalf("expected segments cleared, got %v", st.Segments)
	}
	if len(st.RGB) != 3 || st.RGB[1] != 255 {
		t.Fatalf("unexpected rgb: %v", st.RGB)
	}
}

func TestStore_SetEffect_SetsColorModeAndClearsSegments(t *testing.T) {
	entity := newEntity()
	store := Bind(entity)
	if err := store.SetSegment(Segment{Index: 0}); err != nil {
		t.Fatalf("SetSegment: %v", err)
	}
	if err := store.SetEffect("rainbow", nil); err != nil {
		t.Fatalf("SetEffect: %v", err)
	}
	st, _ := store.Desired()
	if st.ColorMode != ColorModeEffect {
		t.Fatalf("expected color_mode=%q, got %q", ColorModeEffect, st.ColorMode)
	}
	if st.Effect != "rainbow" {
		t.Fatalf("expected effect=rainbow, got %q", st.Effect)
	}
	if len(st.Segments) != 0 {
		t.Fatalf("expected segments cleared, got %v", st.Segments)
	}
}

func TestStore_SetEffect_WithSpeed(t *testing.T) {
	store := Bind(newEntity())
	speed := 3
	if err := store.SetEffect("pulse", &speed); err != nil {
		t.Fatalf("SetEffect: %v", err)
	}
	st, _ := store.Desired()
	if st.EffectSpeed != 3 {
		t.Fatalf("expected effect_speed=3, got %d", st.EffectSpeed)
	}
}

func TestStore_SetSegment_TwoIndexes(t *testing.T) {
	store := Bind(newEntity())
	if err := store.SetSegment(Segment{Index: 0, RGB: []int{255, 0, 0}}); err != nil {
		t.Fatalf("SetSegment 0: %v", err)
	}
	if err := store.SetSegment(Segment{Index: 1, RGB: []int{0, 0, 255}}); err != nil {
		t.Fatalf("SetSegment 1: %v", err)
	}
	st, _ := store.Desired()
	if len(st.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(st.Segments))
	}
	if st.ColorMode != ColorModeSegment {
		t.Fatalf("expected color_mode=%q, got %q", ColorModeSegment, st.ColorMode)
	}
	if st.Segments[0].Index != 0 || st.Segments[1].Index != 1 {
		t.Fatalf("unexpected segment indexes: %v", st.Segments)
	}
}

func TestStore_SetSegment_Upsert(t *testing.T) {
	store := Bind(newEntity())
	if err := store.SetSegment(Segment{Index: 0, RGB: []int{255, 0, 0}}); err != nil {
		t.Fatalf("SetSegment first: %v", err)
	}
	// Update same index
	if err := store.SetSegment(Segment{Index: 0, RGB: []int{0, 255, 0}}); err != nil {
		t.Fatalf("SetSegment upsert: %v", err)
	}
	st, _ := store.Desired()
	if len(st.Segments) != 1 {
		t.Fatalf("expected 1 segment after upsert, got %d", len(st.Segments))
	}
	if st.Segments[0].RGB[1] != 255 {
		t.Fatalf("expected updated rgb, got %v", st.Segments[0].RGB)
	}
}

func TestStore_ClearSegments(t *testing.T) {
	store := Bind(newEntity())
	if err := store.SetSegment(Segment{Index: 0}); err != nil {
		t.Fatalf("SetSegment: %v", err)
	}
	if err := store.ClearSegments(); err != nil {
		t.Fatalf("ClearSegments: %v", err)
	}
	st, _ := store.Desired()
	if st.Segments != nil {
		t.Fatalf("expected segments nil, got %v", st.Segments)
	}
	if st.ColorMode != "" {
		t.Fatalf("expected color_mode cleared, got %q", st.ColorMode)
	}
}

func TestStore_SetReportedFromEvent_RoundTrip(t *testing.T) {
	entity := newEntity()
	store := Bind(entity)

	brightness := 60
	rgb := []int{10, 20, 30}
	if err := store.SetReportedFromEvent(Event{
		Type:       ActionTurnOn,
		Brightness: &brightness,
		RGB:        &rgb,
	}); err != nil {
		t.Fatalf("SetReportedFromEvent: %v", err)
	}

	reported, err := store.Reported()
	if err != nil {
		t.Fatalf("Reported: %v", err)
	}
	if !reported.Power || reported.Brightness != 60 {
		t.Fatalf("unexpected reported state: %+v", reported)
	}
	if len(reported.RGB) != 3 || reported.RGB[0] != 10 {
		t.Fatalf("unexpected reported rgb: %v", reported.RGB)
	}

	// Effective must mirror reported
	var effective State
	if err := json.Unmarshal(entity.Data.Effective, &effective); err != nil {
		t.Fatalf("decode effective: %v", err)
	}
	if effective.Brightness != reported.Brightness || effective.Power != reported.Power {
		t.Fatalf("effective did not mirror reported")
	}
}

func TestStore_SetReportedFromEvent_Segment(t *testing.T) {
	store := Bind(newEntity())
	seg := Segment{Index: 2, RGB: []int{0, 128, 255}}
	if err := store.SetReportedFromEvent(Event{Type: ActionSetSegment, Segment: &seg}); err != nil {
		t.Fatalf("SetReportedFromEvent: %v", err)
	}
	reported, _ := store.Reported()
	if len(reported.Segments) != 1 || reported.Segments[0].Index != 2 {
		t.Fatalf("unexpected segments: %v", reported.Segments)
	}
	if reported.ColorMode != ColorModeSegment {
		t.Fatalf("expected color_mode=segment, got %q", reported.ColorMode)
	}
}

func TestStore_Supports(t *testing.T) {
	entity := &types.Entity{Actions: []string{ActionTurnOn, ActionSetRGB}}
	store := Bind(entity)
	if !store.Supports(ActionTurnOn) {
		t.Fatal("expected Supports(turn_on)=true")
	}
	if !store.Supports(ActionSetRGB) {
		t.Fatal("expected Supports(set_rgb)=true")
	}
	if store.Supports(ActionSetEffect) {
		t.Fatal("expected Supports(set_effect)=false")
	}
}

func TestStore_EnsureDefaultActions(t *testing.T) {
	entity := newEntity()
	store := Bind(entity)
	store.EnsureDefaultActions()
	if !store.Supports(ActionTurnOn) || !store.Supports(ActionClearSegments) {
		t.Fatalf("expected all default actions, got: %v", entity.Actions)
	}
	// Should not override existing actions
	entity.Actions = []string{ActionTurnOff}
	store.EnsureDefaultActions()
	if len(entity.Actions) != 1 || entity.Actions[0] != ActionTurnOff {
		t.Fatalf("expected existing actions preserved, got: %v", entity.Actions)
	}
}

func TestStore_PropagatesDecodeErrors(t *testing.T) {
	badJSON := json.RawMessage(`{"power":`)

	entityDesired := &types.Entity{Data: types.EntityData{Desired: badJSON}}
	if err := Bind(entityDesired).SetDesiredFromCommand(Command{Type: ActionTurnOn}); err == nil {
		t.Fatal("expected desired decode error but got nil")
	}

	entityReported := &types.Entity{Data: types.EntityData{Reported: badJSON}}
	if err := Bind(entityReported).SetReportedFromEvent(Event{Type: ActionTurnOff}); err == nil {
		t.Fatal("expected reported decode error but got nil")
	}
}

func TestStore_SetDesiredFromCommandPreservesOtherFields(t *testing.T) {
	initial := State{
		Power:      true,
		Brightness: 10,
		RGB:        []int{1, 2, 3},
		Effect:     "rainbow",
	}
	b, _ := json.Marshal(initial)
	entity := &types.Entity{Data: types.EntityData{Desired: b}}
	store := Bind(entity)

	brightness := 80
	if err := store.SetDesiredFromCommand(Command{Type: ActionSetBrightness, Brightness: &brightness}); err != nil {
		t.Fatalf("SetDesiredFromCommand: %v", err)
	}
	got, _ := store.Desired()
	if got.Power != initial.Power || got.Brightness != 80 || got.Effect != initial.Effect {
		t.Fatalf("unexpected desired state: %+v", got)
	}
}
