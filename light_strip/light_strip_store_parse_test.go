package light_strip

import (
	"encoding/json"
	"testing"

	types "github.com/slidebolt/sdk-types"
)

func TestParseCommandInvalidJSON(t *testing.T) {
	if _, err := ParseCommand(types.Command{Payload: json.RawMessage(`{`)}); err == nil {
		t.Fatal("expected ParseCommand to fail for malformed json")
	}
}

func TestParseEventInvalidJSON(t *testing.T) {
	if _, err := ParseEvent(types.Event{Payload: json.RawMessage(`{`)}); err == nil {
		t.Fatal("expected ParseEvent to fail for malformed json")
	}
}

func TestParseCommand_ValidPayloads(t *testing.T) {
	brightness := 50
	rgb := []int{255, 0, 128}
	effect := "fire"
	speed := 3

	cases := []struct {
		name    string
		payload string
		check   func(t *testing.T, c Command)
	}{
		{
			name:    "turn_on",
			payload: `{"type":"turn_on"}`,
			check:   func(t *testing.T, c Command) {},
		},
		{
			name:    "turn_off",
			payload: `{"type":"turn_off"}`,
			check:   func(t *testing.T, c Command) {},
		},
		{
			name:    "clear_segments",
			payload: `{"type":"clear_segments"}`,
			check:   func(t *testing.T, c Command) {},
		},
		{
			name:    "set_brightness",
			payload: `{"type":"set_brightness","brightness":50}`,
			check: func(t *testing.T, c Command) {
				if c.Brightness == nil || *c.Brightness != brightness {
					t.Errorf("expected brightness=%d", brightness)
				}
			},
		},
		{
			name:    "set_rgb",
			payload: `{"type":"set_rgb","rgb":[255,0,128]}`,
			check: func(t *testing.T, c Command) {
				if c.RGB == nil || (*c.RGB)[0] != rgb[0] {
					t.Errorf("unexpected rgb: %v", c.RGB)
				}
			},
		},
		{
			name:    "set_effect",
			payload: `{"type":"set_effect","effect":"fire","effect_speed":3}`,
			check: func(t *testing.T, c Command) {
				if c.Effect == nil || *c.Effect != effect {
					t.Errorf("expected effect=%q", effect)
				}
				if c.EffectSpeed == nil || *c.EffectSpeed != speed {
					t.Errorf("expected effect_speed=%d", speed)
				}
			},
		},
		{
			name:    "set_segment",
			payload: `{"type":"set_segment","segment":{"index":2,"rgb":[0,255,0]}}`,
			check: func(t *testing.T, c Command) {
				if c.Segment == nil || c.Segment.Index != 2 {
					t.Errorf("unexpected segment: %v", c.Segment)
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := ParseCommand(types.Command{Payload: json.RawMessage(tc.payload)})
			if err != nil {
				t.Fatalf("ParseCommand failed: %v", err)
			}
			tc.check(t, cmd)
		})
	}
}

func TestParseCommand_InvalidPayloads(t *testing.T) {
	cases := []struct {
		name    string
		payload string
	}{
		{name: "unknown_type", payload: `{"type":"fly"}`},
		{name: "brightness_missing", payload: `{"type":"set_brightness"}`},
		{name: "brightness_out_of_range", payload: `{"type":"set_brightness","brightness":200}`},
		{name: "rgb_wrong_length", payload: `{"type":"set_rgb","rgb":[1,2]}`},
		{name: "effect_empty", payload: `{"type":"set_effect","effect":""}`},
		{name: "segment_missing", payload: `{"type":"set_segment"}`},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseCommand(types.Command{Payload: json.RawMessage(tc.payload)})
			if err == nil {
				t.Fatal("expected error but got nil")
			}
		})
	}
}

func TestParseEvent_ValidPayloads(t *testing.T) {
	cases := []struct {
		name    string
		payload string
	}{
		{name: "turn_on", payload: `{"type":"turn_on"}`},
		{name: "turn_off", payload: `{"type":"turn_off"}`},
		{name: "clear_segments", payload: `{"type":"clear_segments"}`},
		{name: "set_brightness", payload: `{"type":"set_brightness","brightness":75}`},
		{name: "set_rgb", payload: `{"type":"set_rgb","rgb":[100,200,50]}`},
		{name: "set_effect", payload: `{"type":"set_effect","effect":"sparkle"}`},
		{name: "set_segment", payload: `{"type":"set_segment","segment":{"index":0}}`},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseEvent(types.Event{Payload: json.RawMessage(tc.payload)}); err != nil {
				t.Fatalf("ParseEvent failed: %v", err)
			}
		})
	}
}

func TestParseEvent_InvalidPayloads(t *testing.T) {
	cases := []struct {
		name    string
		payload string
	}{
		{name: "unknown_type", payload: `{"type":"glitter"}`},
		{name: "brightness_missing", payload: `{"type":"set_brightness"}`},
		{name: "rgb_bad_component", payload: `{"type":"set_rgb","rgb":[0,0,256]}`},
		{name: "effect_empty", payload: `{"type":"set_effect","effect":""}`},
		{name: "segment_missing", payload: `{"type":"set_segment"}`},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseEvent(types.Event{Payload: json.RawMessage(tc.payload)}); err == nil {
				t.Fatal("expected error but got nil")
			}
		})
	}
}
