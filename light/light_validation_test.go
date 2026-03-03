package light

import "testing"

func TestValidateCommandBrightnessBounds(t *testing.T) {
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
			cmd := Command{Type: ActionSetBrightness, Brightness: &tc.val}
			err := ValidateCommand(cmd)
			if tc.ok && err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("expected error but got nil")
			}
		})
	}
}

func TestValidateEventBrightnessBounds(t *testing.T) {
	v := 101
	err := ValidateEvent(Event{Type: ActionSetBrightness, Brightness: &v})
	if err == nil {
		t.Fatal("expected out-of-range event brightness to fail")
	}
}
