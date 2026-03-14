package light_strip

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

func TestCommandsFromState_PowerOnNoBrightness(t *testing.T) {
	cmds := CommandsFromState(State{Power: true})
	if len(cmds) != 1 {
		t.Errorf("expected 1 command for power-only state, got %d: %v", len(cmds), cmds)
	}
}

func TestCommandsFromState_PowerOnWithBrightness(t *testing.T) {
	cmds := CommandsFromState(State{Power: true, Brightness: 75})
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[0].Type != ActionTurnOn {
		t.Errorf("expected turn_on first, got %q", cmds[0].Type)
	}
	if cmds[1].Type != ActionSetBrightness || *cmds[1].Brightness != 75 {
		t.Errorf("expected set_brightness 75, got %+v", cmds[1])
	}
}

func TestCommandsFromState_RGB(t *testing.T) {
	st := State{Power: true, Brightness: 50, RGB: []int{255, 0, 0}, ColorMode: ColorModeRGB}
	cmds := CommandsFromState(st)
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands, got %d: %v", len(cmds), cmds)
	}
	if cmds[0].Type != ActionTurnOn {
		t.Errorf("expected turn_on first")
	}
	if cmds[1].Type != ActionSetBrightness {
		t.Errorf("expected set_brightness second")
	}
	if cmds[2].Type != ActionSetRGB {
		t.Errorf("expected set_rgb third, got %q", cmds[2].Type)
	}
	if len(*cmds[2].RGB) != 3 || (*cmds[2].RGB)[0] != 255 {
		t.Errorf("unexpected rgb: %v", *cmds[2].RGB)
	}
}

func TestCommandsFromState_Effect(t *testing.T) {
	st := State{Power: true, Effect: "rainbow", ColorMode: ColorModeEffect}
	cmds := CommandsFromState(st)
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d: %v", len(cmds), cmds)
	}
	if cmds[1].Type != ActionSetEffect || *cmds[1].Effect != "rainbow" {
		t.Errorf("expected set_effect rainbow, got %+v", cmds[1])
	}
}

func TestCommandsFromState_EffectWithSpeed(t *testing.T) {
	st := State{Power: true, Effect: "pulse", EffectSpeed: 5, ColorMode: ColorModeEffect}
	cmds := CommandsFromState(st)
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	cmd := cmds[1]
	if cmd.Type != ActionSetEffect {
		t.Fatalf("expected set_effect, got %q", cmd.Type)
	}
	if *cmd.Effect != "pulse" {
		t.Errorf("expected effect=pulse, got %q", *cmd.Effect)
	}
	if cmd.EffectSpeed == nil || *cmd.EffectSpeed != 5 {
		t.Errorf("expected effect_speed=5, got %v", cmd.EffectSpeed)
	}
}

func TestCommandsFromState_Segments(t *testing.T) {
	st := State{
		Power:     true,
		ColorMode: ColorModeSegment,
		Segments: []Segment{
			{Index: 0, RGB: []int{255, 0, 0}},
			{Index: 1, RGB: []int{0, 255, 0}},
			{Index: 2, RGB: []int{0, 0, 255}},
		},
	}
	cmds := CommandsFromState(st)
	// turn_on + 3x set_segment
	if len(cmds) != 4 {
		t.Fatalf("expected 4 commands, got %d: %v", len(cmds), cmds)
	}
	if cmds[0].Type != ActionTurnOn {
		t.Errorf("expected turn_on first")
	}
	for i := 1; i <= 3; i++ {
		if cmds[i].Type != ActionSetSegment {
			t.Errorf("cmd[%d]: expected set_segment, got %q", i, cmds[i].Type)
		}
		if cmds[i].Segment.Index != i-1 {
			t.Errorf("cmd[%d]: expected segment index %d, got %d", i, i-1, cmds[i].Segment.Index)
		}
	}
}

func TestCommandsFromState_PowerOff_NoAttributes(t *testing.T) {
	cmds := CommandsFromState(State{Power: false})
	if len(cmds) != 1 || cmds[0].Type != ActionTurnOff {
		t.Fatalf("expected single turn_off, got %v", cmds)
	}
}

func TestCommandsFromState_RGBMutationSafe(t *testing.T) {
	rgb := []int{1, 2, 3}
	st := State{Power: true, RGB: rgb, ColorMode: ColorModeRGB}
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

func TestStateToCommands_ColorModeRGB(t *testing.T) {
	payloads, err := types.StateToCommands(Type, json.RawMessage(`{"power":true,"rgb":[255,0,0],"color_mode":"rgb"}`))
	if err != nil {
		t.Fatalf("StateToCommands failed: %v", err)
	}
	if len(payloads) != 2 {
		t.Fatalf("expected 2 payloads, got %d", len(payloads))
	}
	var last Command
	if err := json.Unmarshal(payloads[1], &last); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if last.Type != ActionSetRGB {
		t.Fatalf("expected set_rgb, got %s", last.Type)
	}
}

func TestStateToCommands_ColorModeEffect(t *testing.T) {
	payloads, err := types.StateToCommands(Type, json.RawMessage(`{"power":true,"effect":"fire","color_mode":"effect"}`))
	if err != nil {
		t.Fatalf("StateToCommands failed: %v", err)
	}
	if len(payloads) != 2 {
		t.Fatalf("expected 2 payloads, got %d", len(payloads))
	}
	var last Command
	if err := json.Unmarshal(payloads[1], &last); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if last.Type != ActionSetEffect {
		t.Fatalf("expected set_effect, got %s", last.Type)
	}
}
