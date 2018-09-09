package roll

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/konkers/mocktwitch"
)

func newTestBot(t *testing.T) (*Bot, *mocktwitch.Twitch) {
	mock, err := mocktwitch.NewTwitch()
	if err != nil {
		t.Fatalf("Can't create mock twitch: %v.", err)
	}

	testConfig := &Config{
		BotUsername: "RollTheRobot",
		Channel:     "testchan",
		ClientID:    "",
		APIOAuth:    "",
		IRCOAuth:    "",
		IRCAddress:  mock.IrcHost,
		AdminUser:   "rock",
	}

	b := NewBot(testConfig)
	if b == nil {
		t.Fatalf("NewBot() returned nil")
	}
	b.apiClient.UrlBase = mock.ApiUrlBase
	return b, mock
}

func newConnectedTestBot(t *testing.T) (*Bot, *mocktwitch.Twitch) {
	b, mock := newTestBot(t)

	errChan := make(chan error)
	connectChan := make(chan bool)

	b.onConnect = func() {
		connectChan <- true
	}

	go func() {
		err := b.Connect()
		log.Println(err)
		errChan <- err
	}()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Got error: %v", err)
		}
	case err := <-mock.Errors:
		if err != nil {
			t.Fatalf("Got error from mock twitch: %v", err)
		}
	case <-time.After(time.Second * 3):
		t.Fatal("no oauth read")
	case <-connectChan:
		// Success case.
	}

	mock.ChannelStatus.Name = "test"
	return b, mock
}

func TestNewBot(t *testing.T) {
	newTestBot(t)
}

func TestBotConnect(t *testing.T) {
	newConnectedTestBot(t)
}

func TestBotCommand(t *testing.T) {
	b, mock := newConnectedTestBot(t)
	//	b.ircClient.LogWriter = os.Stdout

	wait := make(chan struct{})
	command := func(ctx interface{}, args []string) error {
		_, ok := ctx.(*CommandContext)
		if !ok {
			t.Error("ctx not a CommandContext")
			return fmt.Errorf("ctx not a CommandContext")
		}
		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
		if args[0] != "this" {
			t.Errorf("Expected args[0] to be \"this\", instead got \"%s\"",
				args[0])
		}
		if args[1] != "is" {
			t.Errorf("Expected args[1] to be \"is\", instead got \"%s\"",
				args[1])
		}
		close(wait)
		return nil
	}

	b.AddCommand("test", "test", command, 0)

	mock.SendMessage("testchan", "testuser", "!test this is")

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("test command not invoked")
	}
}

func TestBotFailedCommand(t *testing.T) {
	b, mock := newConnectedTestBot(t)
	//	b.ircClient.LogWriter = os.Stdout

	wait := make(chan struct{})
	command := func(ctx interface{}, args []string) error {
		// TODO(konkers): need a way to verify the error propagates.
		close(wait)
		return fmt.Errorf("error")
	}

	b.AddCommand("test", "test", command, 0)

	mock.SendMessage("testchan", "testuser", "!test")

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("test command not invoked")
	}
}
