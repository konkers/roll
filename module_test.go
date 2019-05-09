package roll

import (
	"fmt"
	"testing"

	"github.com/asdine/storm"
)

var testModuleStarted bool

type testModule struct {
}

func (m *testModule) Start() error {

	testModuleStarted = true
	return nil
}

func (m *testModule) Stop() error {
	return nil
}

func newTestModule(bot *Bot, db storm.Node) (Module, error) {
	return &testModule{}, nil
}

func newBadModule(bot *Bot, db storm.Node) (Module, error) {
	return nil, fmt.Errorf("Bad Module")
}

func TestRegisterAddModuleFactory(t *testing.T) {
	// moduleFactories is global.  This creates potential races for concurrent tests.
	// This is ignored for now but I may want to revisit the whole idea in the future.

	err := RegisterModuleFactory(newTestModule, "test")
	if err != nil {
		t.Errorf("Unexpected error from RegisterModuleFactory(): %v", err)
	}

	err = RegisterModuleFactory(newTestModule, "test")
	if err == nil {
		t.Errorf("Registering duplicate module did not produce an error")
	}

	err = RegisterModuleFactory(newBadModule, "bad")
	if err != nil {
		t.Errorf("Unexpected error from RegisterModuleFactory(): %v", err)
	}

	bot, mock := newTestBot(t)

	err = bot.AddModule("test")
	if err != nil {
		t.Errorf("Unexpected error from AddModule(): %v", err)
	}

	err = bot.AddModule("test")
	if err == nil {
		t.Errorf("Adding duplicate module did not produce an error")
	}

	err = bot.AddModule("test2")
	if err == nil {
		t.Errorf("Adding an unknown module did not produce an error")
	}

	err = bot.AddModule("bad")
	if err == nil {
		t.Errorf("Adding bad module did not produce an error")
	}

	if len(bot.modules) != 1 {
		t.Errorf("%d modules registered successfully.  Expected 1.", len(bot.modules))
	}
	connectTestBot(t, bot, mock)
	if testModuleStarted != true {
		t.Errorf("test module start callback not called.")
	}
}
