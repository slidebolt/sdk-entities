package light

import (
	"encoding/json"
	"sync"
	"testing"

	types "github.com/slidebolt/sdk-types"
)

// === ROUND-TRIP INTEGRATION TESTS ===

func TestStateRoundTrip_PowerOnly(t *testing.T) {
	original := State{Power: true}
	cmds := CommandsFromState(original)

	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	for _, cmd := range cmds {
		if err := store.SetDesiredFromCommand(cmd); err != nil {
			t.Fatalf("SetDesiredFromCommand failed: %v", err)
		}
	}

	result, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}

	if result.Power != original.Power {
		t.Errorf("power: got %v want %v", result.Power, original.Power)
	}
}

func TestStateRoundTrip_FullState(t *testing.T) {
	original := State{
		Power:       true,
		Brightness:  75,
		RGB:         []int{255, 128, 64},
		Temperature: 3000,
		Scene:       "movie",
	}
	cmds := CommandsFromState(original)

	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	for _, cmd := range cmds {
		if err := store.SetDesiredFromCommand(cmd); err != nil {
			t.Fatalf("SetDesiredFromCommand failed: %v", err)
		}
	}

	result, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}

	if result.Power != original.Power {
		t.Errorf("power: got %v want %v", result.Power, original.Power)
	}
	if result.Brightness != original.Brightness {
		t.Errorf("brightness: got %d want %d", result.Brightness, original.Brightness)
	}
	if len(result.RGB) != 3 || result.RGB[0] != original.RGB[0] || result.RGB[1] != original.RGB[1] || result.RGB[2] != original.RGB[2] {
		t.Errorf("rgb: got %v want %v", result.RGB, original.RGB)
	}
	if result.Temperature != original.Temperature {
		t.Errorf("temperature: got %d want %d", result.Temperature, original.Temperature)
	}
	if result.Scene != original.Scene {
		t.Errorf("scene: got %q want %q", result.Scene, original.Scene)
	}
}

func TestStateRoundTrip_PowerOffWithAttributes(t *testing.T) {
	// Power should be first command, then attributes applied
	original := State{
		Power:      false,
		Brightness: 50,
		RGB:        []int{100, 100, 100},
	}
	cmds := CommandsFromState(original)

	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	for _, cmd := range cmds {
		if err := store.SetDesiredFromCommand(cmd); err != nil {
			t.Fatalf("SetDesiredFromCommand failed: %v", err)
		}
	}

	result, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}

	if result.Power {
		t.Errorf("power should be false, got true")
	}
	if result.Brightness != 50 {
		t.Errorf("brightness should persist even when off: got %d want 50", result.Brightness)
	}
}

// === SEMANTIC STORE METHOD TESTS ===

func TestStoreTurnOn(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	if err := store.TurnOn(); err != nil {
		t.Fatalf("TurnOn failed: %v", err)
	}

	state, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}

	if !state.Power {
		t.Errorf("expected power to be true after TurnOn, got false")
	}
}

func TestStoreTurnOff(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// First turn on
	if err := store.TurnOn(); err != nil {
		t.Fatalf("TurnOn failed: %v", err)
	}

	// Then turn off
	if err := store.TurnOff(); err != nil {
		t.Fatalf("TurnOff failed: %v", err)
	}

	state, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}

	if state.Power {
		t.Errorf("expected power to be false after TurnOff, got true")
	}
}

func TestStoreSetBrightness(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	if err := store.SetBrightness(42); err != nil {
		t.Fatalf("SetBrightness failed: %v", err)
	}

	state, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}

	if state.Brightness != 42 {
		t.Errorf("expected brightness 42, got %d", state.Brightness)
	}
}

func TestStoreSetRGB(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	if err := store.SetRGB(255, 128, 64); err != nil {
		t.Fatalf("SetRGB failed: %v", err)
	}

	state, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}

	if len(state.RGB) != 3 {
		t.Fatalf("expected RGB slice of length 3, got %v", state.RGB)
	}
	if state.RGB[0] != 255 || state.RGB[1] != 128 || state.RGB[2] != 64 {
		t.Errorf("expected RGB [255, 128, 64], got %v", state.RGB)
	}
}

func TestStoreSetTemperature(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	if err := store.SetTemperature(4500); err != nil {
		t.Fatalf("SetTemperature failed: %v", err)
	}

	state, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}

	if state.Temperature != 4500 {
		t.Errorf("expected temperature 4500, got %d", state.Temperature)
	}
}

func TestStoreSetScene(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	if err := store.SetScene("reading"); err != nil {
		t.Fatalf("SetScene failed: %v", err)
	}

	state, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}

	if state.Scene != "reading" {
		t.Errorf("expected scene 'reading', got %q", state.Scene)
	}
}

// === NIL POINTER SAFETY TESTS ===

func TestSetDesiredFromCommand_NilBrightnessReturnsError(t *testing.T) {
	// This test documents that SetDesiredFromCommand now validates input
	// and returns an error instead of panicking on nil pointer dereference.
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	// Command with nil Brightness pointer
	cmd := Command{Type: ActionSetBrightness, Brightness: nil}

	if err := store.SetDesiredFromCommand(cmd); err == nil {
		t.Error("expected error from nil pointer, but none occurred")
	}
}

func TestSetDesiredFromCommand_NilRGBReturnsError(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	cmd := Command{Type: ActionSetRGB, RGB: nil}

	if err := store.SetDesiredFromCommand(cmd); err == nil {
		t.Error("expected error from nil RGB, but none occurred")
	}
}

func TestSetDesiredFromCommand_NilTemperatureReturnsError(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	cmd := Command{Type: ActionSetTemperature, Temperature: nil}

	if err := store.SetDesiredFromCommand(cmd); err == nil {
		t.Error("expected error from nil Temperature, but none occurred")
	}
}

func TestSetDesiredFromCommand_NilSceneReturnsError(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	cmd := Command{Type: ActionSetScene, Scene: nil}

	if err := store.SetDesiredFromCommand(cmd); err == nil {
		t.Error("expected error from nil Scene, but none occurred")
	}
}

// === CONCURRENT ACCESS TESTS ===

func TestConcurrentStoreAccess(t *testing.T) {
	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	var wg sync.WaitGroup
	numGoroutines := 100
	iterations := 50

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				brightness := (id*iterations + j) % 101
				if err := store.SetBrightness(brightness); err != nil {
					t.Errorf("goroutine %d iteration %d: SetBrightness failed: %v", id, j, err)
				}
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if _, err := store.Desired(); err != nil {
					t.Errorf("goroutine %d iteration %d: Desired failed: %v", id, j, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is valid JSON
	_, err := store.Desired()
	if err != nil {
		t.Fatalf("final Desired() failed: %v", err)
	}
}

// === PARTIAL STATE UPDATE TESTS ===

func TestPartialUpdate_BrightnessPreservesOtherFields(t *testing.T) {
	initial := State{
		Power:       true,
		Brightness:  10,
		RGB:         []int{1, 2, 3},
		Temperature: 2700,
		Scene:       "relax",
	}
	initialJSON, _ := json.Marshal(initial)

	entity := &types.Entity{Data: types.EntityData{Desired: initialJSON}}
	store := Bind(entity)

	// Update only brightness
	if err := store.SetBrightness(99); err != nil {
		t.Fatalf("SetBrightness failed: %v", err)
	}

	result, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}

	// Verify brightness changed
	if result.Brightness != 99 {
		t.Errorf("brightness should be 99, got %d", result.Brightness)
	}

	// Verify other fields preserved
	if !result.Power {
		t.Errorf("power should remain true, got false")
	}
	if len(result.RGB) != 3 || result.RGB[0] != 1 || result.RGB[1] != 2 || result.RGB[2] != 3 {
		t.Errorf("RGB should be preserved as [1,2,3], got %v", result.RGB)
	}
	if result.Temperature != 2700 {
		t.Errorf("temperature should be 2700, got %d", result.Temperature)
	}
	if result.Scene != "relax" {
		t.Errorf("scene should be 'relax', got %q", result.Scene)
	}
}

func TestPartialUpdate_RGBPreservesOtherFields(t *testing.T) {
	initial := State{
		Power:       true,
		Brightness:  50,
		RGB:         []int{100, 100, 100},
		Temperature: 3000,
		Scene:       "focus",
	}
	initialJSON, _ := json.Marshal(initial)

	entity := &types.Entity{Data: types.EntityData{Desired: initialJSON}}
	store := Bind(entity)

	// Update only RGB
	if err := store.SetRGB(255, 0, 0); err != nil {
		t.Fatalf("SetRGB failed: %v", err)
	}

	result, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}

	// Verify RGB changed
	if len(result.RGB) != 3 || result.RGB[0] != 255 || result.RGB[1] != 0 || result.RGB[2] != 0 {
		t.Errorf("RGB should be [255,0,0], got %v", result.RGB)
	}

	// Verify other fields preserved
	if result.Brightness != 50 {
		t.Errorf("brightness should be 50, got %d", result.Brightness)
	}
	if result.Temperature != 3000 {
		t.Errorf("temperature should be 3000, got %d", result.Temperature)
	}
	if result.Scene != "focus" {
		t.Errorf("scene should be 'focus', got %q", result.Scene)
	}
}

func TestPartialUpdate_ScenePreservesRGBAndBrightness(t *testing.T) {
	initial := State{
		Power:      true,
		Brightness: 75,
		RGB:        []int{255, 128, 0},
		Scene:      "old",
	}
	initialJSON, _ := json.Marshal(initial)

	entity := &types.Entity{Data: types.EntityData{Desired: initialJSON}}
	store := Bind(entity)

	// Update only scene
	if err := store.SetScene("new"); err != nil {
		t.Fatalf("SetScene failed: %v", err)
	}

	result, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired failed: %v", err)
	}

	// Verify scene changed
	if result.Scene != "new" {
		t.Errorf("scene should be 'new', got %q", result.Scene)
	}

	// Verify RGB and brightness preserved
	if result.Brightness != 75 {
		t.Errorf("brightness should be 75, got %d", result.Brightness)
	}
	if len(result.RGB) != 3 || result.RGB[0] != 255 {
		t.Errorf("RGB should be preserved, got %v", result.RGB)
	}
}

// === EDGE CASE TESTS ===

func TestEmptyJSONState(t *testing.T) {
	// Empty JSON should decode to zero values
	entity := &types.Entity{Data: types.EntityData{
		Desired: json.RawMessage(`{}`),
	}}
	store := Bind(entity)

	state, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired with empty JSON failed: %v", err)
	}

	if state.Power {
		t.Errorf("power should be false for empty state, got true")
	}
	if state.Brightness != 0 {
		t.Errorf("brightness should be 0, got %d", state.Brightness)
	}
	if len(state.RGB) != 0 {
		t.Errorf("RGB should be empty, got %v", state.RGB)
	}
}

func TestNullValuesInJSON(t *testing.T) {
	// JSON with explicit null values
	entity := &types.Entity{Data: types.EntityData{
		Desired: json.RawMessage(`{"brightness": null, "rgb": null}`),
	}}
	store := Bind(entity)

	state, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired with null values failed: %v", err)
	}

	if state.Brightness != 0 {
		t.Errorf("null brightness should decode to 0, got %d", state.Brightness)
	}
	if len(state.RGB) != 0 {
		t.Errorf("null RGB should decode to empty slice, got %v", state.RGB)
	}
}

func TestPartialJSONState(t *testing.T) {
	// JSON with only some fields present
	initial := State{
		Power:       true,
		Brightness:  100,
		RGB:         []int{1, 2, 3},
		Temperature: 5000,
		Scene:       "full",
	}
	initialJSON, _ := json.Marshal(initial)

	entity := &types.Entity{Data: types.EntityData{Desired: initialJSON}}
	store := Bind(entity)

	// Now overwrite with partial JSON
	entity.Data.Desired = json.RawMessage(`{"power": false}`)

	state, err := store.Desired()
	if err != nil {
		t.Fatalf("Desired with partial JSON failed: %v", err)
	}

	// Only power should change, others should be zero (not preserved from previous)
	if state.Power {
		t.Errorf("power should be false, got true")
	}
	// Note: In Go JSON unmarshaling, unspecified fields keep their zero value
	// This is actually a potential design issue - partial updates via JSON
	// don't preserve previous state like command updates do
}

// === MUTATION SAFETY TESTS ===

func TestSetReportedFromEvent_MutationSafety(t *testing.T) {
	rgb := []int{10, 20, 30}
	evt := Event{
		Type: ActionSetRGB,
		RGB:  &rgb,
	}

	entity := &types.Entity{Data: types.EntityData{}}
	store := Bind(entity)

	if err := store.SetReportedFromEvent(evt); err != nil {
		t.Fatalf("SetReportedFromEvent failed: %v", err)
	}

	// Mutate original slice
	rgb[0] = 99

	// Verify reported state wasn't affected
	reported, err := store.Reported()
	if err != nil {
		t.Fatalf("Reported failed: %v", err)
	}

	if reported.RGB[0] == 99 {
		t.Error("SetReportedFromEvent RGB slice is not a clone — mutation propagated to reported state")
	}

	// Also verify effective state wasn't affected
	effective, err := decodeState(entity.Data.Effective)
	if err != nil {
		t.Fatalf("decode effective failed: %v", err)
	}

	if effective.RGB[0] == 99 {
		t.Error("SetReportedFromEvent RGB slice is not a clone — mutation propagated to effective state")
	}
}

// === INIT FUNCTION TESTS ===

func TestDomainRegistration(t *testing.T) {
	// The init() function runs automatically, but we can verify the side effects
	// by checking that types package has our domain registered
	// This test assumes sdk-types provides a way to query registered domains
	// If not available, we verify via the fact that tests can import and use the package

	// Verify that Describe() returns valid domain info
	desc := Describe()
	if desc.Domain != Type {
		t.Errorf("domain name mismatch: got %q want %q", desc.Domain, Type)
	}
	if len(desc.Commands) == 0 {
		t.Error("expected at least one command in domain descriptor")
	}
	if len(desc.Events) == 0 {
		t.Error("expected at least one event in domain descriptor")
	}
}

// === COMMAND VALIDATION EDGE CASES ===

func TestValidateCommand_UnsupportedAction(t *testing.T) {
	cmd := Command{Type: "unsupported_action"}
	err := ValidateCommand(cmd)
	if err == nil {
		t.Error("expected error for unsupported action, got nil")
	}
}

func TestValidateEvent_UnsupportedAction(t *testing.T) {
	evt := Event{Type: "unsupported_action"}
	err := ValidateEvent(evt)
	if err == nil {
		t.Error("expected error for unsupported action, got nil")
	}
}

func TestValidateCommand_BrightnessBoundaryValues(t *testing.T) {
	cases := []struct {
		name string
		val  int
		want bool
	}{
		{"exactly_zero", 0, true},
		{"exactly_100", 100, true},
		{"negative_one", -1, false},
		{"one_hundred_one", 101, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := Command{Type: ActionSetBrightness, Brightness: &tc.val}
			err := ValidateCommand(cmd)
			got := err == nil
			if got != tc.want {
				t.Errorf("ValidateCommand(brightness=%d): got valid=%v want valid=%v", tc.val, got, tc.want)
			}
		})
	}
}
