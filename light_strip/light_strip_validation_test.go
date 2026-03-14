package light_strip

import "testing"

func TestValidateCommand_AllTypes(t *testing.T) {
	brightness50 := 50
	rgb := []int{255, 128, 0}
	effect := "rainbow"
	seg := Segment{Index: 0, RGB: []int{255, 0, 0}, Brightness: 50}

	cases := []struct {
		name string
		cmd  Command
		ok   bool
	}{
		{name: "turn_on", cmd: Command{Type: ActionTurnOn}, ok: true},
		{name: "turn_off", cmd: Command{Type: ActionTurnOff}, ok: true},
		{name: "clear_segments", cmd: Command{Type: ActionClearSegments}, ok: true},
		{name: "set_brightness_ok", cmd: Command{Type: ActionSetBrightness, Brightness: &brightness50}, ok: true},
		{name: "set_rgb_ok", cmd: Command{Type: ActionSetRGB, RGB: &rgb}, ok: true},
		{name: "set_effect_ok", cmd: Command{Type: ActionSetEffect, Effect: &effect}, ok: true},
		{name: "set_segment_ok", cmd: Command{Type: ActionSetSegment, Segment: &seg}, ok: true},
		{name: "unknown_type", cmd: Command{Type: "blink"}, ok: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCommand(tc.cmd)
			if tc.ok && err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("expected error but got nil")
			}
		})
	}
}

func TestValidateCommand_BrightnessBounds(t *testing.T) {
	cases := []struct {
		name string
		val  int
		ok   bool
	}{
		{name: "min", val: 0, ok: true},
		{name: "mid", val: 55, ok: true},
		{name: "max", val: 100, ok: true},
		{name: "below", val: -1, ok: false},
		{name: "above", val: 101, ok: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCommand(Command{Type: ActionSetBrightness, Brightness: &tc.val})
			if tc.ok && err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("expected error but got nil")
			}
		})
	}
}

func TestValidateCommand_BrightnessRequired(t *testing.T) {
	if err := ValidateCommand(Command{Type: ActionSetBrightness}); err == nil {
		t.Fatal("expected error for missing brightness")
	}
}

func TestValidateCommand_RGBWrongLength(t *testing.T) {
	bad := []int{1, 2}
	if err := ValidateCommand(Command{Type: ActionSetRGB, RGB: &bad}); err == nil {
		t.Fatal("expected error for rgb length != 3")
	}
}

func TestValidateCommand_RGBComponentOutOfBounds(t *testing.T) {
	bad := []int{0, 0, 256}
	if err := ValidateCommand(Command{Type: ActionSetRGB, RGB: &bad}); err == nil {
		t.Fatal("expected error for rgb component > 255")
	}
	neg := []int{-1, 0, 0}
	if err := ValidateCommand(Command{Type: ActionSetRGB, RGB: &neg}); err == nil {
		t.Fatal("expected error for negative rgb component")
	}
}

func TestValidateCommand_EffectEmpty(t *testing.T) {
	empty := ""
	if err := ValidateCommand(Command{Type: ActionSetEffect, Effect: &empty}); err == nil {
		t.Fatal("expected error for empty effect")
	}
}

func TestValidateCommand_EffectTooLong(t *testing.T) {
	long := make([]byte, MaxEffectLength+1)
	for i := range long {
		long[i] = 'a'
	}
	s := string(long)
	if err := ValidateCommand(Command{Type: ActionSetEffect, Effect: &s}); err == nil {
		t.Fatal("expected error for effect name too long")
	}
}

func TestValidateCommand_SegmentRequired(t *testing.T) {
	if err := ValidateCommand(Command{Type: ActionSetSegment}); err == nil {
		t.Fatal("expected error for missing segment")
	}
}

func TestValidateCommand_SegmentNegativeIndex(t *testing.T) {
	seg := Segment{Index: -1}
	if err := ValidateCommand(Command{Type: ActionSetSegment, Segment: &seg}); err == nil {
		t.Fatal("expected error for negative segment index")
	}
}

func TestValidateCommand_SegmentRGBInvalid(t *testing.T) {
	seg := Segment{Index: 0, RGB: []int{1, 2}}
	if err := ValidateCommand(Command{Type: ActionSetSegment, Segment: &seg}); err == nil {
		t.Fatal("expected error for segment rgb wrong length")
	}
	seg2 := Segment{Index: 0, RGB: []int{0, 0, 300}}
	if err := ValidateCommand(Command{Type: ActionSetSegment, Segment: &seg2}); err == nil {
		t.Fatal("expected error for segment rgb component out of bounds")
	}
}

func TestValidateCommand_SegmentBrightnessOutOfRange(t *testing.T) {
	seg := Segment{Index: 0, Brightness: 101}
	if err := ValidateCommand(Command{Type: ActionSetSegment, Segment: &seg}); err == nil {
		t.Fatal("expected error for segment brightness > 100")
	}
}

func TestValidateEvent_BrightnessBounds(t *testing.T) {
	v := 101
	if err := ValidateEvent(Event{Type: ActionSetBrightness, Brightness: &v}); err == nil {
		t.Fatal("expected out-of-range event brightness to fail")
	}
}

func TestValidateEvent_AllTypes(t *testing.T) {
	brightness50 := 50
	rgb := []int{0, 128, 255}
	effect := "fire"
	seg := Segment{Index: 1}

	cases := []struct {
		name string
		evt  Event
		ok   bool
	}{
		{name: "turn_on", evt: Event{Type: ActionTurnOn}, ok: true},
		{name: "turn_off", evt: Event{Type: ActionTurnOff}, ok: true},
		{name: "clear_segments", evt: Event{Type: ActionClearSegments}, ok: true},
		{name: "set_brightness_ok", evt: Event{Type: ActionSetBrightness, Brightness: &brightness50}, ok: true},
		{name: "set_brightness_missing", evt: Event{Type: ActionSetBrightness}, ok: false},
		{name: "set_rgb_ok", evt: Event{Type: ActionSetRGB, RGB: &rgb}, ok: true},
		{name: "set_rgb_bad_len", evt: Event{Type: ActionSetRGB, RGB: func() *[]int { v := []int{1, 2}; return &v }()}, ok: false},
		{name: "set_effect_ok", evt: Event{Type: ActionSetEffect, Effect: &effect}, ok: true},
		{name: "set_effect_empty", evt: Event{Type: ActionSetEffect, Effect: func() *string { v := ""; return &v }()}, ok: false},
		{name: "set_segment_ok", evt: Event{Type: ActionSetSegment, Segment: &seg}, ok: true},
		{name: "set_segment_missing", evt: Event{Type: ActionSetSegment}, ok: false},
		{name: "unknown_type", evt: Event{Type: "disco"}, ok: false},
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
