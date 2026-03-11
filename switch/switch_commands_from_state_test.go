package entityswitch

import "testing"

func TestCommandsFromState_On(t *testing.T) {
	cmds := CommandsFromState(State{Power: true})
	if len(cmds) != 1 || cmds[0].Type != ActionTurnOn {
		t.Fatalf("expected [turn_on], got %v", cmds)
	}
}

func TestCommandsFromState_Off(t *testing.T) {
	cmds := CommandsFromState(State{Power: false})
	if len(cmds) != 1 || cmds[0].Type != ActionTurnOff {
		t.Fatalf("expected [turn_off], got %v", cmds)
	}
}
