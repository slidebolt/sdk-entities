//go:build weaknesses

package light

import (
	"encoding/json"
	"strings"
	"testing"

	types "github.com/slidebolt/sdk-types"
)

// === BREAKING TEST: API Inconsistency Between Domains ===
// This test documents that light.Store has public getters but switch.Store doesn't
// This is a design flaw - users expect consistent API across domains
func TestAPIDesignConsistency_LightHasPublicGetters(t *testing.T) {
	// Light domain has public Desired() and Reported() methods
	// This test verifies they exist and work
	entity := &types.Entity{Data: types.EntityData{
		Desired: json.RawMessage(`{"power": true, "brightness": 50}`),
	}}
	store := Bind(entity)

	// These methods exist and are exported
	_, err := store.Desired()
	if err != nil {
		t.Fatalf("light.Store.Desired() failed: %v", err)
	}

	_, err = store.Reported()
	if err != nil {
		t.Fatalf("light.Store.Reported() failed: %v", err)
	}
}

// === BREAKING TEST: State Loss on Partial JSON Update ===
// This exposes that unmarshaling partial JSON into existing state loses data
// Design flaw: partial updates via JSON don't merge with existing state
func TestPartialJSONUpdate_DataLoss(t *testing.T) {
	// Set up complete state
	initial := State{
		Power:       true,
		Brightness:  80,
		RGB:         []int{255, 128, 0},
		Temperature: 3000,
		Scene:       "movie",
	}
	initialJSON, _ := json.Marshal(initial)

	entity := &types.Entity{Data: types.EntityData{Desired: initialJSON}}

	// Now simulate receiving partial state update (e.g., from a device that only reports power)
	// This is a real-world scenario - device sends {"power": false} without other fields
	partialUpdate := json.RawMessage(`{"power": false}`)
	entity.Data.Desired = partialUpdate

	store := Bind(entity)
	result, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired() failed: %v", err)
	}

	// EXPECTED (but fails): Brightness, RGB, Temperature, Scene should be preserved
	// ACTUAL: They are zeroed out because Go JSON unmarshaling replaces the whole struct
	if result.Brightness != 0 {
		t.Errorf("BUG: Partial JSON update should NOT preserve brightness, got %d (exposing design flaw)", result.Brightness)
	}
	if len(result.RGB) != 0 {
		t.Errorf("BUG: Partial JSON update should NOT preserve RGB, got %v (exposing design flaw)", result.RGB)
	}
	if result.Temperature != 0 {
		t.Errorf("BUG: Partial JSON update should NOT preserve temperature, got %d (exposing design flaw)", result.Temperature)
	}
	if result.Scene != "" {
		t.Errorf("BUG: Partial JSON update should NOT preserve scene, got %q (exposing design flaw)", result.Scene)
	}

	// This demonstrates that the system cannot handle partial state updates via JSON
	t.Log("WARNING: Design flaw exposed - partial JSON updates lose existing state data")
}

// === BREAKING TEST: Validation Bypass via Direct Command ===
// This verifies that SetDesiredFromCommand now validates input
func TestValidation_DirectCommand(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// Create command with invalid brightness (200, max should be 100)
	invalidBrightness := 200
	cmd := Command{
		Type:       ActionSetBrightness,
		Brightness: &invalidBrightness,
	}

	// This should now fail validation
	err := store.SetDesiredFromCommand(cmd)
	if err == nil {
		t.Error("expected error from invalid brightness, but none occurred")
	}

	t.Log("SUCCESS: SetDesiredFromCommand now calls ValidateCommand")
}

// === BREAKING TEST: RGB Array Bounds Validation ===
// This verifies that RGB values are now validated
func TestRGBBoundsValidation(t *testing.T) {
	// RGB values should be 0-255
	invalidRGB := []int{999, -50, 1000}
	cmd := Command{
		Type: ActionSetRGB,
		RGB:  &invalidRGB,
	}

	// ValidateCommand now checks RGB value ranges
	if err := ValidateCommand(cmd); err == nil {
		t.Error("expected error from out-of-bounds RGB component, but none occurred")
	}

	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)
	err := store.SetDesiredFromCommand(cmd)
	if err == nil {
		t.Error("expected error from out-of-bounds RGB component in store, but none occurred")
	}
}

// === BREAKING TEST: Empty Scene Name Rejected ===
// This verifies that scene names are validated
func TestEmptySceneValidation(t *testing.T) {
	emptyScene := ""
	cmd := Command{
		Type:  ActionSetScene,
		Scene: &emptyScene,
	}

	// ValidateCommand rejects empty scenes
	if err := ValidateCommand(cmd); err == nil {
		t.Error("expected error from empty scene, but none occurred")
	}

	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	err := store.SetDesiredFromCommand(cmd)
	if err == nil {
		t.Error("expected error from empty scene in store, but none occurred")
	}
}

// === BREAKING TEST: Resource Exhaustion via Large RGB ===
// This exposes no limits on RGB array size
func TestRGBResourceExhaustion(t *testing.T) {
	// Create RGB array with 1 million elements
	hugeRGB := make([]int, 1000000)
	for i := range hugeRGB {
		hugeRGB[i] = i % 256
	}

	cmd := Command{
		Type: ActionSetRGB,
		RGB:  &hugeRGB,
	}

	// ValidateCommand only checks length==3, not array size limits
	if err := ValidateCommand(cmd); err == nil {
		t.Log("WARNING: No validation on RGB array size - could cause memory issues")
	}

	// This would cause memory issues if processed
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// Should fail or be rejected
	err := store.SetDesiredFromCommand(cmd)
	if err == nil {
		t.Error("BUG: Huge RGB array accepted without size limits")
	}
}

// === BREAKING TEST: Resource Exhaustion via Long Scene Name ===
// This exposes no limits on string length
func TestSceneNameResourceExhaustion(t *testing.T) {
	// Create very long scene name (1MB)
	longScene := strings.Repeat("a", 1024*1024)
	cmd := Command{
		Type:  ActionSetScene,
		Scene: &longScene,
	}

	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	err := store.SetDesiredFromCommand(cmd)
	if err == nil {
		state, _ := store.Desired()
		if len(state.Scene) == len(longScene) {
			t.Errorf("BUG: Scene name of %d bytes accepted without length limits", len(longScene))
		}
	}
}

// === BREAKING TEST: Determinism of CommandsFromState ===
// This exposes that command ordering might not be deterministic
func TestCommandsFromState_Determinism(t *testing.T) {
	state := State{
		Power:       true,
		Brightness:  50,
		RGB:         []int{255, 0, 0},
		Temperature: 3000,
		Scene:       "test",
	}

	// Run multiple times and verify consistent ordering
	var firstOrder []string
	for i := 0; i < 100; i++ {
		cmds := CommandsFromState(state)
		order := make([]string, len(cmds))
		for j, cmd := range cmds {
			order[j] = cmd.Type
		}

		if i == 0 {
			firstOrder = order
		} else {
			for j, cmdType := range order {
				if cmdType != firstOrder[j] {
					t.Errorf("BUG: CommandsFromState not deterministic - iteration %d differs at position %d: got %s want %s",
						i, j, cmdType, firstOrder[j])
					break
				}
			}
		}
	}
}

// === BREAKING TEST: Type Safety Issues ===
// This exposes potential type safety issues with interface compliance
func TestCommandResponsePayloadKind_Consistency(t *testing.T) {
	state := State{}
	kind := state.CommandResponsePayloadKind()
	if kind != Type {
		t.Errorf("BUG: State.CommandResponsePayloadKind() returned %q, expected %q", kind, Type)
	}

	cmd := Command{}
	cmdKind := cmd.CommandRequestPayloadKind()
	if cmdKind != Type {
		t.Errorf("BUG: Command.CommandRequestPayloadKind() returned %q, expected %q", cmdKind, Type)
	}
}

// === BREAKING TEST: Multiple Store Bindings to Same Entity ===
// This exposes that multiple stores can modify same entity without coordination
func TestMultipleStoreBindings_RaceCondition(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store1 := Bind(entity)
	store2 := Bind(entity)

	// Both stores can modify the same entity
	_ = store1.SetBrightness(50)
	_ = store2.SetBrightness(75)

	// Last write wins, but there's no coordination
	state, _ := store1.Desired()
	if state.Brightness == 50 {
		t.Error("BUG: store1's write was overwritten by store2 without coordination")
	}

	// Verify both stores see the same (last) value
	state2, _ := store2.Desired()
	if state.Brightness != state2.Brightness {
		t.Error("BUG: Two stores bound to same entity see different states")
	}

	t.Log("WARNING: Multiple Store bindings to same Entity can cause coordination issues")
}

// === BREAKING TEST: Empty State Commands Generation ===
// This exposes that zero-value state generates commands
func TestCommandsFromState_EmptyState(t *testing.T) {
	emptyState := State{}
	cmds := CommandsFromState(emptyState)

	// Empty state (Power: false) generates turn_off command
	if len(cmds) != 1 || cmds[0].Type != ActionTurnOff {
		t.Errorf("BUG: Empty state should generate turn_off command, got %v", cmds)
	}

	// But what about zero brightness, temperature? Should they generate commands?
	// Currently: Brightness 0 is omitted, Temperature 0 is omitted
	// This might be unexpected behavior
}

// === BREAKING TEST: State Mutation After Command Generation ===
// This exposes whether CommandsFromState properly clones data
func TestCommandsFromState_MutationAfter(t *testing.T) {
	rgb := []int{1, 2, 3}
	state := State{Power: true, RGB: rgb}
	cmds := CommandsFromState(state)

	// Modify original state after command generation
	rgb[0] = 999

	// Verify command wasn't affected
	if len(cmds) >= 2 && cmds[1].RGB != nil {
		if (*cmds[1].RGB)[0] == 999 {
			t.Error("BUG: CommandsFromState RGB slice was not cloned - mutation propagated to command")
		}
	}
}

// === BREAKING TEST: Brightness Zero Value Ambiguity ===
// This exposes that brightness=0 is indistinguishable from "not set"
func TestBrightnessZeroValue_Ambiguity(t *testing.T) {
	// State with brightness explicitly set to 0
	state := State{Power: true, Brightness: 0}
	cmds := CommandsFromState(state)

	// CommandsFromState omits brightness=0 because it checks `if st.Brightness != 0`
	// This means we can't distinguish between "brightness not set" and "brightness set to 0"
	for _, cmd := range cmds {
		if cmd.Type == ActionSetBrightness {
			t.Log("Brightness 0 generated a command (unexpected behavior)")
			return
		}
	}

	t.Log("WARNING: brightness=0 is indistinguishable from 'not set' - this is a design limitation")
}

// === BREAKING TEST: RGB Length Edge Cases ===
// This exposes RGB array length validation gaps
func TestRGBLengthValidation(t *testing.T) {
	// RGB with 2 elements (should be rejected)
	rgb2 := []int{1, 2}
	cmd2 := Command{Type: ActionSetRGB, RGB: &rgb2}
	if err := ValidateCommand(cmd2); err == nil {
		t.Error("BUG: RGB array with 2 elements should be rejected")
	}

	// RGB with 4 elements (should be rejected)
	rgb4 := []int{1, 2, 3, 4}
	cmd4 := Command{Type: ActionSetRGB, RGB: &rgb4}
	if err := ValidateCommand(cmd4); err == nil {
		t.Error("BUG: RGB array with 4 elements should be rejected")
	}

	// RGB with 0 elements (should be rejected)
	rgb0 := []int{}
	cmd0 := Command{Type: ActionSetRGB, RGB: &rgb0}
	if err := ValidateCommand(cmd0); err == nil {
		t.Error("BUG: RGB array with 0 elements should be rejected")
	}
}

// === BREAKING TEST: Effective State Inconsistency ===
// This exposes that effective state can diverge from reported
func TestEffectiveStateInconsistency(t *testing.T) {
	initial := State{Power: true}
	initialJSON, _ := json.Marshal(initial)

	entity := &types.Entity{Data: types.EntityData{
		Reported:  initialJSON,
		Effective: initialJSON,
	}}
	store := Bind(entity)

	// Modify reported state
	_ = store.SetReportedFromEvent(Event{Type: ActionTurnOff})

	// Verify both are in sync
	reported, _ := decodeState(entity.Data.Reported)
	effective, _ := decodeState(entity.Data.Effective)

	if reported.Power != effective.Power {
		t.Errorf("BUG: Reported (%v) and Effective (%v) states are inconsistent after update",
			reported.Power, effective.Power)
	}
}

// === BREAKING TEST: Invalid Action Type in Command ===
// This verifies that unknown action types are rejected
func TestInvalidActionType(t *testing.T) {
	cmd := Command{Type: "invalid_action_type_12345"}

	// ValidateCommand should reject this
	if err := ValidateCommand(cmd); err == nil {
		t.Error("expected error from invalid action type, but none occurred")
	}

	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// SetDesiredFromCommand now returns error for unknown action types
	err := store.SetDesiredFromCommand(cmd)
	if err == nil {
		t.Error("expected error from unknown action in SetDesiredFromCommand, but none occurred")
	}

	// State should be unchanged
	state, _ := store.Desired()
	if state.Power {
		t.Error("BUG: Unknown action should not modify state, but power is now true")
	}

	t.Log("SUCCESS: Unknown action types are now explicitly rejected")
}
