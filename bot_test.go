package roll

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/konkers/mocktwitch"
	"github.com/phayes/freeport"
)

func newTestBot(t *testing.T) (*Bot, *mocktwitch.Twitch) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	mock, err := mocktwitch.NewTwitch()
	if err != nil {
		t.Fatalf("Can't create mock twitch: %v.", err)
	}

	tmpFile, err := ioutil.TempFile("", "bot.*.db")
	if err != nil {
		t.Fatalf("Can't get temporary file: %v", err)
	}
	dbPath := tmpFile.Name()

	httpsPort, err := freeport.GetFreePort()
	if err != nil {
		t.Fatalf("Can't get free port for https server: %v", err)
	}

	httpPort, err := freeport.GetFreePort()
	if err != nil {
		t.Fatalf("Can't get free port for http server: %v", err)
	}

	testConfig := &Config{
		BotUsername: "RollTheRobot",
		Channel:     "testchan",
		ClientID:    "C012345678abcdefg",
		APIOAuth:    "A012345678abcdefg",
		IRCOAuth:    "I012345678abcdefg",
		IRCAddress:  mock.IrcHost,
		APIURLBase:  mock.ApiUrlBase,
		HTTPSAddr:   fmt.Sprintf("localhost:%d", httpsPort),
		HTTPAddr:    fmt.Sprintf("localhost:%d", httpPort),
		KeyFile:     mock.Keys.KeyFilename,
		CertFile:    mock.Keys.CertFilename,
		AdminUser:   "rock",
		DBPath:      dbPath,
	}

	b, err := NewBot(testConfig)
	if b == nil {
		t.Fatalf("NewBot() returned nil")
	}
	if err != nil {
		t.Fatalf("NewBot() returned error: %v", err)
	}
	b.apiClient.UrlBase = mock.ApiUrlBase
	return b, mock
}

func connectTestBot(t *testing.T, b *Bot, mock *mocktwitch.Twitch) {

	err := b.Connect()
	if err != nil {
		t.Fatalf("Got error: %v", err)
	}

	mock.ChannelStatus.Name = "test"
}

func newConnectedTestBot(t *testing.T) (*Bot, *mocktwitch.Twitch) {
	b, mock := newTestBot(t)
	connectTestBot(t, b, mock)
	return b, mock
}

func TestNewBot(t *testing.T) {
	newTestBot(t)
}

func TestIrcConnectError(t *testing.T) {
	b, _ := newTestBot(t)
	b.ircClient.IrcAddress = "ldskfjsdlfjksdlf:2343"
	err := b.Connect()
	if err == nil {
		t.Fatalf("Expected error connecting to garbage IRC address")
	}
}

func TestIrcConnectTimeout(t *testing.T) {
	b, mock := newTestBot(t)
	mock.SquelchIrc = true
	err := b.Connect()
	if err == nil {
		t.Fatalf("Expected error connecting to squelched IRC server")
	}
}

func TestNewBotDefaults(t *testing.T) {
	testConfig := &Config{}
	bot, err := NewBot(testConfig)
	if err != nil {
		t.Fatalf("NewBot() returned error: %v", err)
	}

	if bot.Config.DBPath != "bot.db" {
		t.Errorf("DBPath default %s is not the expected bot.db.", bot.Config.DBPath)
	}
}

func TestNewBotBadDbPath(t *testing.T) {
	testConfig := &Config{
		DBPath: "/",
	}
	_, err := NewBot(testConfig)
	if err == nil {
		t.Fatalf("NewBot() didn't return an error with invalid DBPath")
	}
}

func TestBotConnect(t *testing.T) {
	newConnectedTestBot(t)
}

func TestBotCommand(t *testing.T) {
	b, mock := newConnectedTestBot(t)

	wait := make(chan struct{})
	command := func(cc *CommandContext, args []string) error {
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

	wait := make(chan struct{})
	command := func(cc *CommandContext, args []string) error {
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

func TestUserLevel(t *testing.T) {
	b, _ := newTestBot(t)

	adminUserLevel := b.userLevel(b.Config.AdminUser)
	if adminUserLevel != 100 {
		t.Errorf("Admin user level(%d) != 100", adminUserLevel)
	}

	normalUserLevel := b.userLevel("nobody")
	if normalUserLevel != 0 {
		t.Errorf("Admin user level(%d) != 0", normalUserLevel)
	}
}

func TestIsAdminRequest(t *testing.T) {
	b, _ := newTestBot(t)
	if b.IsAdminRequest(nil) != true {
		t.Errorf("nil request is not an admin request")
	}

	req, err := http.NewRequest("GET", "http://localhost/test", nil)
	if err != nil {
		t.Fatalf("Can't create request: %v", err)
	}

	if b.IsAdminRequest(req) == true {
		t.Errorf("Empty admin request IS an admin request")
	}

	req.Header.Set("Client-ID", b.Config.ClientID)
	if b.IsAdminRequest(req) == true {
		t.Errorf("Request with only Client-ID IS an admin request")
	}
	req.Header.Del("Client-ID")

	req.Header.Set("Authorization", "OAuth "+b.Config.APIOAuth)
	if b.IsAdminRequest(req) == true {
		t.Errorf("Request with only Authorization IS an admin request")
	}
	req.Header.Set("Client-ID", b.Config.ClientID)

	if b.IsAdminRequest(req) != true {
		t.Errorf("Request with Client ID AND Authorization IS NOT an admin request")
	}
}

func TestAccessors(t *testing.T) {
	b, _ := newTestBot(t)

	if b.Irc() != b.ircClient {
		t.Errorf("Irc() accessor did not return ircClient")
	}
}
