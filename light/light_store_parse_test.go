package light

import (
	"encoding/json"
	"testing"

	types "github.com/slidebolt/sdk-types"
)

func TestParseCommandAndEventInvalidJSON(t *testing.T) {
	if _, err := ParseCommand(types.Command{Payload: json.RawMessage(`{`)}); err == nil {
		t.Fatal("expected ParseCommand to fail for malformed json")
	}
	if _, err := ParseEvent(types.Event{Payload: json.RawMessage(`{`)}); err == nil {
		t.Fatal("expected ParseEvent to fail for malformed json")
	}
}

func TestValidateEventActionPayloads(t *testing.T) {
	brightness := 50
	badRGB := []int{1, 2}
	goodRGB := []int{1, 2, 3}
	temperature := 3000
	scene := "movie"
	emptyScene := ""

	cases := []struct {
		name string
		evt  Event
		ok   bool
	}{
		{name: "turn_on", evt: Event{Type: ActionTurnOn}, ok: true},
		{name: "set_brightness_missing", evt: Event{Type: ActionSetBrightness}, ok: false},
		{name: "set_brightness_ok", evt: Event{Type: ActionSetBrightness, Brightness: &brightness}, ok: true},
		{name: "set_rgb_bad_len", evt: Event{Type: ActionSetRGB, RGB: &badRGB}, ok: false},
		{name: "set_rgb_ok", evt: Event{Type: ActionSetRGB, RGB: &goodRGB}, ok: true},
		{name: "set_temperature_missing", evt: Event{Type: ActionSetTemperature}, ok: false},
		{name: "set_temperature_ok", evt: Event{Type: ActionSetTemperature, Temperature: &temperature}, ok: true},
		{name: "set_scene_empty", evt: Event{Type: ActionSetScene, Scene: &emptyScene}, ok: false},
		{name: "set_scene_ok", evt: Event{Type: ActionSetScene, Scene: &scene}, ok: true},
		{name: "unsupported", evt: Event{Type: "unknown"}, ok: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateEvent(tc.evt)
			if tc.ok && err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("expected error but got nil")
			}
		})
	}
}

func TestStoreSetDesiredFromCommandPreservesState(t *testing.T) {
	initial := State{
		Power:       true,
		Brightness:  10,
		RGB:         []int{1, 2, 3},
		Temperature: 2700,
		Scene:       "relax",
	}
	initialDesired, err := json.Marshal(initial)
	if err != nil {
		t.Fatalf("marshal initial state: %v", err)
	}
	entity := &types.Entity{Data: types.EntityData{Desired: initialDesired}}
	store := Bind(entity)

	brightness := 80
	if err := store.SetDesiredFromCommand(Command{Type: ActionSetBrightness, Brightness: &brightness}); err != nil {
		t.Fatalf("SetDesiredFromCommand failed: %v", err)
	}

	got, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}
	if got.Power != initial.Power || got.Brightness != 80 || got.Temperature != initial.Temperature || got.Scene != initial.Scene {
		t.Fatalf("unexpected desired state: %+v", got)
	}
}

func TestStoreSetReportedFromEventWritesEffective(t *testing.T) {
	initial := State{
		Power:       false,
		Brightness:  25,
		RGB:         []int{10, 20, 30},
		Temperature: 3000,
		Scene:       "relax",
	}
	initialReported, err := json.Marshal(initial)
	if err != nil {
		t.Fatalf("marshal initial state: %v", err)
	}
	entity := &types.Entity{Data: types.EntityData{Reported: initialReported}}
	store := Bind(entity)

	scene := "reading"
	if err := store.SetReportedFromEvent(Event{Type: ActionSetScene, Scene: &scene}); err != nil {
		t.Fatalf("SetReportedFromEvent failed: %v", err)
	}

	reported, err := store.Reported()
	if err != nil {
		t.Fatalf("Reported failed: %v", err)
	}
	if reported.Scene != "reading" || reported.Brightness != initial.Brightness {
		t.Fatalf("unexpected reported state: %+v", reported)
	}

	effective, err := decodeState(entity.Data.Effective)
	if err != nil {
		t.Fatalf("decode effective failed: %v", err)
	}
	if effective.Scene != reported.Scene || effective.Brightness != reported.Brightness {
		t.Fatalf("effective did not mirror reported: reported=%+v effective=%+v", reported, effective)
	}
}

func TestStorePropagatesDecodeErrors(t *testing.T) {
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

func TestStoreEnsureDefaultActionsAndSupports(t *testing.T) {
	entity := &types.Entity{}
	store := Bind(entity)
	store.EnsureDefaultActions()
	if !store.Supports(ActionTurnOn) || !store.Supports(ActionSetScene) {
		t.Fatalf("expected default light actions, got: %v", entity.Actions)
	}

	entity.Actions = []string{ActionTurnOff}
	store.EnsureDefaultActions()
	if len(entity.Actions) != 1 || entity.Actions[0] != ActionTurnOff {
		t.Fatalf("expected existing actions to be preserved, got: %v", entity.Actions)
	}
}
