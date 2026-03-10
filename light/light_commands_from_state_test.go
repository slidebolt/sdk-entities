package light

import (
	"encoding/json"
	"testing"

	types "github.com/slidebolt/sdk-types"
)

func TestCommandsFromState_PowerOn(t *testing.T) {
	cmds := CommandsFromState(State{Power: true})
	if len(cmds) == 0 || cmds[0].Type != ActionTurnOn {
		t.Fatalf("expected first command to be turn_on, got %v", cmds)
	}
}

func TestCommandsFromState_PowerOff(t *testing.T) {
	cmds := CommandsFromState(State{Power: false})
	if len(cmds) == 0 || cmds[0].Type != ActionTurnOff {
		t.Fatalf("expected first command to be turn_off, got %v", cmds)
	}
}

func TestCommandsFromState_PowerFirst(t *testing.T) {
	cmds := CommandsFromState(State{Power: true, Brightness: 80})
	if cmds[0].Type != ActionTurnOn {
		t.Errorf("expected power command first, got %q", cmds[0].Type)
	}
}

func TestCommandsFromState_ZeroFieldsOmitted(t *testing.T) {
	cmds := CommandsFromState(State{Power: true})
	if len(cmds) != 1 {
		t.Errorf("expected 1 command for power-only state, got %d: %v", len(cmds), cmds)
	}
}

func TestCommandsFromState_AllFields(t *testing.T) {
	rgb := []int{255, 128, 0}
	st := State{
		Power:       true,
		Brightness:  60,
		RGB:         rgb,
		Temperature: 3000,
		Scene:       "relax",
	}
	cmds := CommandsFromState(st)

	wantTypes := []string{
		ActionTurnOn,
		ActionSetBrightness,
		ActionSetRGB,
		ActionSetTemperature,
		ActionSetScene,
	}
	if len(cmds) != len(wantTypes) {
		t.Fatalf("expected %d commands, got %d: %v", len(wantTypes), len(cmds), cmds)
	}
	for i, want := range wantTypes {
		if cmds[i].Type != want {
			t.Errorf("cmd[%d]: got %q want %q", i, cmds[i].Type, want)
		}
	}

	if *cmds[1].Brightness != 60 {
		t.Errorf("brightness: got %d want 60", *cmds[1].Brightness)
	}
	if len(*cmds[2].RGB) != 3 || (*cmds[2].RGB)[0] != 255 {
		t.Errorf("rgb: got %v", *cmds[2].RGB)
	}
	if *cmds[3].Temperature != 3000 {
		t.Errorf("temperature: got %d want 3000", *cmds[3].Temperature)
	}
	if *cmds[4].Scene != "relax" {
		t.Errorf("scene: got %q want %q", *cmds[4].Scene, "relax")
	}
}

func TestCommandsFromState_RGBMutationSafe(t *testing.T) {
	rgb := []int{1, 2, 3}
	st := State{Power: true, RGB: rgb}
	cmds := CommandsFromState(st)
	rgb[0] = 99
	if (*cmds[1].RGB)[0] == 99 {
		t.Error("CommandsFromState RGB slice is not a clone — mutation propagated")
	}
}

func TestStateToCommands_BrightnessZeroPreservedWhenPresent(t *testing.T) {
	payloads, err := types.StateToCommands(Type, json.RawMessage(`{"power":true,"brightness":0}`))
	if err != nil {
		t.Fatalf("StateToCommands failed: %v", err)
	}
	if len(payloads) != 2 {
		t.Fatalf("expected 2 payloads (turn_on + set_brightness), got %d", len(payloads))
	}
	var cmd Command
	if err := json.Unmarshal(payloads[1], &cmd); err != nil {
		t.Fatalf("decode brightness payload: %v", err)
	}
	if cmd.Type != ActionSetBrightness || cmd.Brightness == nil || *cmd.Brightness != 0 {
		t.Fatalf("unexpected brightness command: %+v", cmd)
	}
}

func TestStateToCommands_ColorModeRGBWinsOverTemperature(t *testing.T) {
	payloads, err := types.StateToCommands(Type, json.RawMessage(`{"power":true,"brightness":20,"rgb":[255,0,0],"temperature":2700,"color_mode":"rgb"}`))
	if err != nil {
		t.Fatalf("StateToCommands failed: %v", err)
	}
	if len(payloads) != 3 {
		t.Fatalf("expected 3 payloads (turn_on + brightness + rgb), got %d", len(payloads))
	}
	var last Command
	if err := json.Unmarshal(payloads[2], &last); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if last.Type != ActionSetRGB {
		t.Fatalf("expected final command set_rgb, got %s", last.Type)
	}
}

func TestStateToCommands_ColorModeTemperatureWinsOverRGB(t *testing.T) {
	payloads, err := types.StateToCommands(Type, json.RawMessage(`{"power":true,"brightness":50,"rgb":[255,0,0],"temperature":2700,"color_mode":"temperature"}`))
	if err != nil {
		t.Fatalf("StateToCommands failed: %v", err)
	}
	if len(payloads) != 3 {
		t.Fatalf("expected 3 payloads (turn_on + brightness + temperature), got %d", len(payloads))
	}
	var last Command
	if err := json.Unmarshal(payloads[2], &last); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if last.Type != ActionSetTemperature {
		t.Fatalf("expected final command set_temperature, got %s", last.Type)
	}
}
