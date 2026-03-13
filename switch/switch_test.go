package entityswitch

import (
	"encoding/json"
	"testing"

	types "github.com/slidebolt/sdk-types"
)

func TestValidateCommandAndEvent(t *testing.T) {
	if err := ValidateCommand(Command{Type: ActionTurnOn}); err != nil {
		t.Fatalf("expected turn_on command to be valid: %v", err)
	}
	if err := ValidateCommand(Command{Type: "bad"}); err == nil {
		t.Fatal("expected unsupported command to fail")
	}
	if err := ValidateEvent(Event{Type: ActionTurnOff}); err != nil {
		t.Fatalf("expected turn_off event to be valid: %v", err)
	}
	if err := ValidateEvent(Event{Type: "bad"}); err == nil {
		t.Fatal("expected unsupported event to fail")
	}
}

func TestParseCommandAndEventInvalidJSON(t *testing.T) {
	if _, err := ParseCommand(types.Command{Payload: json.RawMessage(`{`)}); err == nil {
		t.Fatal("expected ParseCommand to fail for malformed json")
	}
	if _, err := ParseEvent(types.Event{Payload: json.RawMessage(`{`)}); err == nil {
		t.Fatal("expected ParseEvent to fail for malformed json")
	}
}

func TestStoreSetDesiredAndReported(t *testing.T) {
	entity := &types.Entity{}
	store := Bind(entity)

	if err := store.SetDesiredFromCommand(Command{Type: ActionTurnOn}); err != nil {
		t.Fatalf("SetDesiredFromCommand failed: %v", err)
	}
	desired, err := store.readDesired()
	if err != nil {
		t.Fatalf("decode desired failed: %v", err)
	}
	if !desired.Power {
		t.Fatalf("expected desired power true, got %+v", desired)
	}

	if err := store.SetReportedFromEvent(Event{Type: ActionTurnOff}); err != nil {
		t.Fatalf("SetReportedFromEvent failed: %v", err)
	}
	reported, err := store.readReported()
	if err != nil {
		t.Fatalf("decode reported failed: %v", err)
	}
	effective, err := decodeState(entity.Data.Effective)
	if err != nil {
		t.Fatalf("decode effective failed: %v", err)
	}
	if reported.Power || effective.Power {
		t.Fatalf("expected reported/effective power false, reported=%+v effective=%+v", reported, effective)
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
	if !store.Supports(ActionTurnOn) || !store.Supports(ActionTurnOff) {
		t.Fatalf("expected default switch actions, got: %v", entity.Actions)
	}

	entity.Actions = []string{ActionTurnOff}
	store.EnsureDefaultActions()
	if len(entity.Actions) != 1 || entity.Actions[0] != ActionTurnOff {
		t.Fatalf("expected existing actions to be preserved, got: %v", entity.Actions)
	}
}
