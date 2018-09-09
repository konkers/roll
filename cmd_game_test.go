package roll

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/konkers/mocktwitch"
)

func drainMessages(mock *mocktwitch.Twitch) {
	for {
		select {
		case _ = <-mock.IrcMeassageChan:
		default:
			return
		}
	}
}

func expectResponse(t *testing.T, mock *mocktwitch.Twitch, response string) {
	lastMsg := ""
	for {
		select {
		case msg := <-mock.IrcMeassageChan:
			msg = strings.TrimPrefix(msg, "PRIVMSG #test :")
			lastMsg = msg
			if msg == response {
				return
			}
		case <-time.After(time.Second):
			t.Errorf("Did not get \"%s\".  Last message: \"%s\".", response, lastMsg)
			return
		}
	}
}

func TestGameCommand(t *testing.T) {
	b, mock := newConnectedTestBot(t)
	b.ircClient.LogWriter = os.Stdout

	newGame := "Mega Man 1"
	mock.ChannelStatus.Game = newGame

	drainMessages(mock)
	mock.SendMessage("test", "rush", "!game")
	expectResponse(t, mock, gameResponse(newGame))
}

func TestSetGameCommand(t *testing.T) {
	b, mock := newConnectedTestBot(t)
	b.ircClient.LogWriter = os.Stdout

	oldGame := "Mega Man 2"
	newGame := "Mega Man 1"

	mock.ChannelStatus.Game = oldGame

	time.Sleep(time.Second)
	drainMessages(mock)

	mock.SendMessage("test", "rush", "!setgame Mega Man 1")
	if b.cmdErr != nil &&
		b.cmdErr.Error() != "Can't exec \"!setgame Mega Man 1\": user level 0 not >= 10." {
		t.Errorf("Did not get expected error: %v", b.cmdErr)
	}

	mock.SendMessage("test", "rock", "!setgame Mega Man 1")
	expectResponse(t, mock, setGameResponse(newGame))
}
