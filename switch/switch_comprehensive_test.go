package entityswitch

import (
	"encoding/json"
	"sync"
	"testing"

	types "github.com/slidebolt/sdk-types"
)

// === ROUND-TRIP INTEGRATION TESTS ===

func TestStateRoundTrip_On(t *testing.T) {
	original := State{Power: true}
	cmds := CommandsFromState(original)

	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	for _, cmd := range cmds {
		if err := store.SetDesiredFromCommand(cmd); err != nil {
			t.Fatalf("SetDesiredFromCommand failed: %v", err)
		}
	}

	// Note: Switch Store doesn't have public Desired() method - using decodeState
	result, err := decodeState(entity.Data.Desired)
	if err != nil {
		t.Fatalf("decode desired failed: %v", err)
	}

	if result.Power != original.Power {
		t.Errorf("power: got %v want %v", result.Power, original.Power)
	}
}

func TestStateRoundTrip_Off(t *testing.T) {
	original := State{Power: false}
	cmds := CommandsFromState(original)

	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	for _, cmd := range cmds {
		if err := store.SetDesiredFromCommand(cmd); err != nil {
			t.Fatalf("SetDesiredFromCommand failed: %v", err)
		}
	}

	result, err := decodeState(entity.Data.Desired)
	if err != nil {
		t.Fatalf("decode desired failed: %v", err)
	}

	if result.Power != original.Power {
		t.Errorf("power: got %v want %v", result.Power, original.Power)
	}
}

// === CONCURRENT ACCESS TESTS ===

func TestConcurrentStoreAccess(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	var wg sync.WaitGroup
	numGoroutines := 50
	iterations := 30

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cmd := Command{Type: ActionTurnOn}
				if (id+j)%2 == 0 {
					cmd = Command{Type: ActionTurnOff}
				}
				if err := store.SetDesiredFromCommand(cmd); err != nil {
					t.Errorf("goroutine %d iteration %d: SetDesiredFromCommand failed: %v", id, j, err)
				}
			}
		}(i)
	}

	// Concurrent reads using private method since no public getter exists
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if _, err := store.readDesired(); err != nil {
					t.Errorf("goroutine %d iteration %d: readDesired failed: %v", id, j, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is valid
	_, err := decodeState(entity.Data.Desired)
	if err != nil {
		t.Fatalf("final decode failed: %v", err)
	}
}

// === API INCONSISTENCY DOCUMENTATION TESTS ===

// TestPublicStateAccess documents the missing public API in switch domain
func TestPublicStateAccess(t *testing.T) {
	// This test documents that unlike light.Store, switch.Store does NOT have
	// public Desired() and Reported() methods. Users must use the private
	// decodeState() helper or access entity.Data directly.

	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// Set some state first
	store.SetDesiredFromCommand(Command{Type: ActionTurnOn})

	// Verify we can access state via decodeState (the workaround)
	state, err := decodeState(entity.Data.Desired)
	if err != nil {
		t.Fatalf("decodeState failed: %v", err)
	}

	if !state.Power {
		t.Error("expected power to be true after TurnOn command")
	}

	// Note: There's no store.Desired() like in light domain
	// This is an API inconsistency between domains
}

// === EDGE CASE TESTS ===

func TestEmptyJSONState(t *testing.T) {
	// Empty JSON should decode to zero values (Power: false)
	entity := &types.Entity{Data: types.EntityData{
		Desired: json.RawMessage(`{}`),
	}}
	store := Bind(entity)

	state, err := store.readDesired()
	if err != nil {
		t.Fatalf("readDesired with empty JSON failed: %v", err)
	}

	if state.Power {
		t.Errorf("power should be false for empty state, got true")
	}
}

func TestMalformedJSON(t *testing.T) {
	// Malformed JSON should return error
	entity := &types.Entity{Data: types.EntityData{
		Desired: json.RawMessage(`{"power":`),
	}}
	store := Bind(entity)

	_, err := store.readDesired()
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

// === STATE PERSISTENCE TESTS ===

func TestDesiredStatePersistence(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// Turn on
	if err := store.SetDesiredFromCommand(Command{Type: ActionTurnOn}); err != nil {
		t.Fatalf("SetDesiredFromCommand(TurnOn) failed: %v", err)
	}

	state1, _ := decodeState(entity.Data.Desired)
	if !state1.Power {
		t.Error("expected power to be true after TurnOn")
	}

	// Turn off
	if err := store.SetDesiredFromCommand(Command{Type: ActionTurnOff}); err != nil {
		t.Fatalf("SetDesiredFromCommand(TurnOff) failed: %v", err)
	}

	state2, _ := decodeState(entity.Data.Desired)
	if state2.Power {
		t.Error("expected power to be false after TurnOff")
	}
}

func TestReportedAndEffectiveState(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// Set reported state via event
	if err := store.SetReportedFromEvent(Event{Type: ActionTurnOn}); err != nil {
		t.Fatalf("SetReportedFromEvent failed: %v", err)
	}

	// Verify reported state
	reported, err := decodeState(entity.Data.Reported)
	if err != nil {
		t.Fatalf("decode reported failed: %v", err)
	}
	if !reported.Power {
		t.Error("expected reported power to be true")
	}

	// Verify effective state mirrors reported
	effective, err := decodeState(entity.Data.Effective)
	if err != nil {
		t.Fatalf("decode effective failed: %v", err)
	}
	if !effective.Power {
		t.Error("expected effective power to be true (should mirror reported)")
	}
}

// === STATE TO COMMAND CONVERSION TESTS ===

func TestCommandsFromState_Ordering(t *testing.T) {
	// For switch, there's only power command, so order is simple
	state := State{Power: true}
	cmds := CommandsFromState(state)

	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}

	if cmds[0].Type != ActionTurnOn {
		t.Errorf("expected command type %q, got %q", ActionTurnOn, cmds[0].Type)
	}
}

// === ERROR PROPAGATION TESTS ===

func TestSetDesiredFromCommand_PropagatesDecodeError(t *testing.T) {
	// Set malformed JSON as initial desired state
	entity := &types.Entity{Data: types.EntityData{
		Desired: json.RawMessage(`{"power":`),
	}}
	store := Bind(entity)

	err := store.SetDesiredFromCommand(Command{Type: ActionTurnOn})
	if err == nil {
		t.Error("expected error when decoding malformed desired state, got nil")
	}
}

func TestSetReportedFromEvent_PropagatesDecodeError(t *testing.T) {
	// Set malformed JSON as initial reported state
	entity := &types.Entity{Data: types.EntityData{
		Reported: json.RawMessage(`{"power":`),
	}}
	store := Bind(entity)

	err := store.SetReportedFromEvent(Event{Type: ActionTurnOff})
	if err == nil {
		t.Error("expected error when decoding malformed reported state, got nil")
	}
}

// === DOMAIN REGISTRATION TEST ===

func TestDomainRegistration(t *testing.T) {
	// Verify that Describe() returns valid domain descriptor
	desc := Describe()

	if desc.Domain != Type {
		t.Errorf("domain name mismatch: got %q want %q", desc.Domain, Type)
	}

	// Switch should have exactly 2 commands: turn_on, turn_off
	if len(desc.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(desc.Commands))
	}

	if len(desc.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(desc.Events))
	}
}
