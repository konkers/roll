package roll

import (
	"testing"
)

func TestVfs(t *testing.T) {
	bot, _ := newTestBot(t)

	// TODO(konkers): verify the contents of the file

	// Try a non archived file.
	_, err := bot.openFile("data/gen.go")
	if err != nil {
		t.Errorf("Can't open non-archived file: %v", err)
	}

	// Try an archived file.
	_, err = bot.openFile("templates/index.html")
	if err != nil {
		t.Errorf("Can't open non-archived file: %v", err)
	}

	// Try a missing
	_, err = bot.openFile("templates/___index.html")
	if err == nil {
		t.Errorf("Expected an error opening a missing file.")
	}
}
