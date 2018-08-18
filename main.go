package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gempir/go-twitch-irc"
	"github.com/konkers/cmd"
	"github.com/konkers/twitchapi"
)

type Config struct {
	BotUsername string
	Channel     string
	ClientID    string
	APIOAuth    string
	IRCOAuth    string
}

type Bot struct {
	Config    *Config
	IRCClient *twitch.Client
	APIClient *twitchapi.Connection
}

type TwitchContext struct {
	Bot     *Bot
	Channel string
	User    *twitch.User
	Message *twitch.Message
}

func userLevel(user string) int {
	if user == "djkonkers" {
		return 100
	} else {
		return 0
	}
}

func gameCommand(ctx interface{}, args []string) error {
	tc, ok := ctx.(*TwitchContext)
	if !ok {
		return fmt.Errorf("ctx not a TwitchContext")
	}

	channel, err := tc.Bot.APIClient.GetChannel()
	if err != nil {
		return err
	}
	log.Printf("%#v", channel)

	tc.Bot.IRCClient.Say(tc.Channel, fmt.Sprintf("The game is %s.", channel.Game))

	return nil
}

func handler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<html><body><a href=\"https://id.twitch.tv/oauth2/authorize?client_id=<CLIENTID>&redirect_uri=https://localhost:10443/auth&response_type=token&scope=channel_editor+channel_subscriptions+channel_read\">Authorize</a></body></html>"))
}

func setGameCommand(ctx interface{}, args []string) error {

	tc, ok := ctx.(*TwitchContext)
	if !ok {
		return fmt.Errorf("ctx not a TwitchContext")
	}

	channel, err := tc.Bot.APIClient.GetChannel()
	if err != nil {
		return err
	}

	game := strings.Join(args, " ")

	err = tc.Bot.APIClient.SetChannelGame(channel.ID, game)
	if err != nil {
		return err
	}
	tc.Bot.IRCClient.Say(tc.Channel, fmt.Sprintf("If I knew how, I'd set the game to %s.", game))

	return nil
}

func main() {
	bot := Bot{
		Config: &Config{
			BotUsername: "RollTheRobot",
			Channel:     "",
			ClientID:    "",
			APIOAuth:    "",
			IRCOAuth:    "",
		},
	}
	bot.APIClient = twitchapi.NewConnection(bot.Config.ClientID, bot.Config.APIOAuth)
	bot.IRCClient = twitch.NewClient(bot.Config.BotUsername, bot.Config.IRCOAuth)

	http.HandleFunc("/", handler)
	log.Printf("About to listen on 10443. Go to https://127.0.0.1:10443/")
	go http.ListenAndServeTLS(":10443", "cert.pem", "key.pem", nil)

	cmd := cmd.NewEngine()
	cmd.AddCommand("game", "Tells the channel the current game.", gameCommand, 0)
	cmd.AddCommand("setgame", "Sets the stream game.", setGameCommand, 10)

	bot.IRCClient.OnNewMessage(func(channel string, user twitch.User, message twitch.Message) {
		fmt.Printf("%s] %#v\n", channel, user)
		fmt.Printf("  %#v\n", message)

		if strings.HasPrefix(message.Text, "!") {
			ctx := &TwitchContext{
				Bot:     &bot,
				Channel: channel,
				User:    &user,
				Message: &message,
			}
			err := cmd.ExecString(ctx, userLevel(user.Username), strings.TrimPrefix(message.Text, "!"))
			if err != nil {
				log.Printf("Can't exec \"%s\": %v.", message.Text, err)
			}
		}
	})

	bot.IRCClient.OnNewUserstateMessage(func(channel string, user twitch.User, message twitch.Message) {
		if message.Tags["mod"] != "1" {
			log.Printf("WARNING: %s does not have op on %s!", message.Tags["display-name"], channel)
		} else {
			log.Printf("%s has op on %s.", message.Tags["display-name"], channel)
		}

	})

	bot.IRCClient.Join(bot.Config.Channel)

	bot.IRCClient.LogWriter = os.Stdout
	bot.IRCClient.LogPrefix = "[TMI] "
	err := bot.IRCClient.Connect()
	if err != nil {
		panic(err)
	}
}
