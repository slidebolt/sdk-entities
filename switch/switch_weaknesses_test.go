//go:build weaknesses

package entityswitch

import (
	"encoding/json"
	"testing"

	types "github.com/slidebolt/sdk-types"
)

// === BREAKING TEST: Missing Public API ===
// This exposes that switch.Store lacks public getter methods that light.Store has
func TestMissingPublicAPI_DesiredMethod(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{
		Desired: json.RawMessage(`{"power": true}`),
	}}
	store := Bind(entity)
	_ = store // Document that we have to use the store to bind, but can't call Desired()

	// Try to access state via public API
	// This should work like light.Store.Desired(), but switch.Store doesn't have it
	// Users must use decodeState() which is a design inconsistency

	state, err := decodeState(entity.Data.Desired)
	if err != nil {
		t.Fatalf("Had to use workaround (decodeState): %v", err)
	}

	if !state.Power {
		t.Error("BUG: Power should be true")
	}

	t.Log("WARNING: switch.Store lacks public Desired()/Reported() methods - inconsistent with light.Store")
}

// === BREAKING TEST: State Loss on Partial JSON ===
// Exposes same design flaw as light domain - partial JSON updates lose data
func TestPartialJSONUpdate_DataLoss(t *testing.T) {
	// Switch only has Power field, but let's verify the same issue exists
	initial := State{Power: true}
	initialJSON, _ := json.Marshal(initial)

	entity := &types.Entity{Data: types.EntityData{
		Desired: initialJSON,
	}}

	// Simulate partial update
	entity.Data.Desired = json.RawMessage(`{}`)

	store := Bind(entity)
	state, err := store.readDesired()
	if err != nil {
		t.Fatalf("readDesired failed: %v", err)
	}

	// Power should be false (zero value) because {} has no power field
	if state.Power {
		t.Error("BUG: Partial JSON {} resulted in true power - this is unexpected behavior")
	}

	t.Log("INFO: Partial JSON correctly zeroes out missing fields (consistent behavior)")
}

// === BREAKING TEST: Validation Bypass ===
// Exposes that SetDesiredFromCommand doesn't validate
func TestValidationBypass_InvalidCommand(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// Invalid command type
	cmd := Command{Type: "nonexistent_action"}

	// SetDesiredFromCommand doesn't validate - it will silently ignore
	err := store.SetDesiredFromCommand(cmd)
	if err != nil {
		t.Fatalf("SetDesiredFromCommand rejected command - unexpected: %v", err)
	}

	// State should be unchanged (which is correct behavior for ignored command)
	state, _ := store.readDesired()
	if state.Power {
		t.Error("BUG: Invalid command modified state")
	}

	// But ValidateCommand should have rejected it
	if err := ValidateCommand(cmd); err == nil {
		t.Error("BUG: ValidateCommand should reject unknown action types")
	}

	t.Log("WARNING: SetDesiredFromCommand doesn't validate - relies on caller to validate first")
}

// === BREAKING TEST: Multiple Store Bindings ===
// Exposes coordination issue when multiple stores bind to same entity
func TestMultipleStoreBindings_CoordinationIssue(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store1 := Bind(entity)
	store2 := Bind(entity)

	// Both stores write to same entity
	store1.SetDesiredFromCommand(Command{Type: ActionTurnOn})
	store2.SetDesiredFromCommand(Command{Type: ActionTurnOff})

	// Last write wins
	state, _ := store1.readDesired()
	if state.Power {
		t.Error("BUG: store1's TurnOn was overwritten by store2's TurnOff - no coordination")
	}

	state2, _ := store2.readDesired()
	if state.Power != state2.Power {
		t.Error("BUG: Stores see different states for same entity")
	}

	t.Log("WARNING: Multiple Store bindings to same Entity lack coordination")
}

// === BREAKING TEST: Empty Command Type ===
// Exposes that empty string as action type is not handled
func TestEmptyCommandType(t *testing.T) {
	cmd := Command{Type: ""}

	// ValidateCommand should reject empty type
	if err := ValidateCommand(cmd); err == nil {
		t.Error("BUG: Empty command type should be rejected")
	}

	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// SetDesiredFromCommand silently ignores empty type
	err := store.SetDesiredFromCommand(cmd)
	if err != nil {
		t.Errorf("BUG: Empty command should be silently ignored, but got error: %v", err)
	}

	t.Log("WARNING: Empty command type is silently ignored")
}

// === BREAKING TEST: Concurrent Access Without Synchronization ===
// Exposes race condition in store operations (light has this too)
func TestConcurrentAccess_Race(t *testing.T) {
	// This is a simpler version that might pass but documents the issue
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// Rapidly toggle state
	for i := 0; i < 1000; i++ {
		if i%2 == 0 {
			store.SetDesiredFromCommand(Command{Type: ActionTurnOn})
		} else {
			store.SetDesiredFromCommand(Command{Type: ActionTurnOff})
		}
	}

	// Verify final state is valid JSON
	_, err := decodeState(entity.Data.Desired)
	if err != nil {
		t.Errorf("BUG: Concurrent-like rapid access corrupted JSON: %v", err)
	}
}

// === BREAKING TEST: Determinism ===
// Verifies CommandsFromState is deterministic
func TestCommandsFromState_Determinism(t *testing.T) {
	state := State{Power: true}

	// Run multiple times
	var firstCmd string
	for i := 0; i < 100; i++ {
		cmds := CommandsFromState(state)
		if len(cmds) != 1 {
			t.Fatalf("Expected 1 command, got %d", len(cmds))
		}

		if i == 0 {
			firstCmd = cmds[0].Type
		} else if cmds[0].Type != firstCmd {
			t.Errorf("BUG: CommandsFromState not deterministic - iteration %d got %s want %s",
				i, cmds[0].Type, firstCmd)
		}
	}
}

// === BREAKING TEST: Type Consistency ===
// Verifies Type constant and Command/Event types match
func TestTypeConsistency(t *testing.T) {
	if Type != "switch" {
		t.Errorf("BUG: Type constant is %q, expected 'switch'", Type)
	}

	state := State{}
	if state.CommandResponsePayloadKind() != Type {
		t.Error("BUG: State.CommandResponsePayloadKind() doesn't match Type constant")
	}

	cmd := Command{}
	if cmd.CommandRequestPayloadKind() != Type {
		t.Error("BUG: Command.CommandRequestPayloadKind() doesn't match Type constant")
	}
}

// === BREAKING TEST: Domain Descriptor Completeness ===
// Verifies Describe() returns complete metadata
func TestDomainDescriptorCompleteness(t *testing.T) {
	desc := Describe()

	if desc.Domain != Type {
		t.Errorf("BUG: Domain mismatch: got %q want %q", desc.Domain, Type)
	}

	// Should have exactly 2 actions for switch
	if len(desc.Commands) != 2 {
		t.Errorf("BUG: Expected 2 commands, got %d", len(desc.Commands))
	}

	if len(desc.Events) != 2 {
		t.Errorf("BUG: Expected 2 events, got %d", len(desc.Events))
	}

	// Verify specific actions exist
	actionMap := make(map[string]bool)
	for _, a := range desc.Commands {
		actionMap[a.Action] = true
	}

	if !actionMap[ActionTurnOn] {
		t.Error("BUG: Domain descriptor missing ActionTurnOn")
	}
	if !actionMap[ActionTurnOff] {
		t.Error("BUG: Domain descriptor missing ActionTurnOff")
	}
}

// === BREAKING TEST: EnsureDefaultActions Idempotency ===
// Verifies calling EnsureDefaultActions multiple times is safe
func TestEnsureDefaultActions_Idempotency(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// First call
	store.EnsureDefaultActions()
	actions1 := len(entity.Actions)

	// Second call (should be no-op)
	store.EnsureDefaultActions()
	actions2 := len(entity.Actions)

	if actions1 != actions2 {
		t.Errorf("BUG: Second EnsureDefaultActions changed action count from %d to %d", actions1, actions2)
	}

	// Verify existing actions preserved
	entity.Actions = []string{ActionTurnOff}
	store.EnsureDefaultActions()

	if len(entity.Actions) != 1 || entity.Actions[0] != ActionTurnOff {
		t.Errorf("BUG: EnsureDefaultActions overwrote existing actions: %v", entity.Actions)
	}
}

// === BREAKING TEST: Supports Method Edge Cases ===
// Tests Supports() with various inputs
func TestSupports_EdgeCases(t *testing.T) {
	entity := &types.Entity{
		Actions: []string{ActionTurnOn},
	}
	store := Bind(entity)

	// Supported action
	if !store.Supports(ActionTurnOn) {
		t.Error("BUG: Supports() returned false for supported action")
	}

	// Unsupported action
	if store.Supports(ActionTurnOff) {
		t.Error("BUG: Supports() returned true for unsupported action")
	}

	// Empty action
	if store.Supports("") {
		t.Error("BUG: Supports() returned true for empty action")
	}

	// Case sensitivity (assuming actions are case-sensitive)
	if store.Supports("turn_on") { // lowercase
		t.Log("INFO: Action matching is case-sensitive (may or may not be desired)")
	}
}

// === BREAKING TEST: SetReportedFromEvent Event Type Validation ===
// Exposes that event types are not validated
func TestSetReportedFromEvent_InvalidEventType(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// Set power on first
	store.SetDesiredFromCommand(Command{Type: ActionTurnOn})

	// Now send invalid event
	invalidEvent := Event{Type: "invalid_type"}
	err := store.SetReportedFromEvent(invalidEvent)
	if err != nil {
		t.Fatalf("SetReportedFromEvent rejected invalid event - unexpected: %v", err)
	}

	// State should be unchanged
	state, _ := store.readReported()
	_ = state // Will be default/empty since we never set reported before
	// The event was silently ignored
	t.Log("WARNING: Invalid event types are silently ignored in SetReportedFromEvent")
}

// === BREAKING TEST: Malformed State Data ===
// Tests behavior when entity has malformed initial state
func TestMalformedInitialState(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{
		Desired:  json.RawMessage(`{invalid json`),
		Reported: json.RawMessage(`{also invalid`),
	}}
	store := Bind(entity)

	// readDesired should fail
	_, err := store.readDesired()
	if err == nil {
		t.Error("BUG: readDesired should fail with malformed JSON")
	}

	// readReported should fail
	_, err = store.readReported()
	if err == nil {
		t.Error("BUG: readReported should fail with malformed JSON")
	}

	// But SetDesiredFromCommand should propagate the error
	err = store.SetDesiredFromCommand(Command{Type: ActionTurnOn})
	if err == nil {
		t.Error("BUG: SetDesiredFromCommand should fail when initial state is malformed")
	}
}

// === BREAKING TEST: Package Name Inconsistency ===
// Documents the package name inconsistency with light package
func TestPackageNameConsistency(t *testing.T) {
	// This test documents that:
	// - Light package is named "light"
	// - Switch package is named "entityswitch" (not "switch")

	// The import path issue:
	// import "github.com/slidebolt/sdk-entities/light"  -> package light
	// import "github.com/slidebolt/sdk-entities/switch" -> package entityswitch

	// This inconsistency can cause confusion
	t.Log("WARNING: Package naming inconsistency - light uses 'light', switch uses 'entityswitch'")
	t.Log("This requires: import entityswitch 'github.com/slidebolt/sdk-entities/switch'")
}
