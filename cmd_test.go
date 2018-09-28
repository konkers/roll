package roll

import "testing"

func TestCmdEngineBadContext(t *testing.T) {
	engine := NewCmdEngine()
	command := func(cc *CommandContext, args []string) error {
		return nil
	}

	engine.AddCommand("test", "test help", command, 0)
	err := engine.Exec(int(1), 10, []string{"test"})
	if err == nil {
		t.Errorf("Did not get expected error when invoking command w/ invalid context")
	}
}
