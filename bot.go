package roll

import (
	"log"
	"net/http"
	"strings"

	"github.com/asdine/storm"
	twitch "github.com/gempir/go-twitch-irc"
	"github.com/konkers/cmd"
	"github.com/konkers/twitchapi"
)

// Bot is the main object that represents the bot.
type Bot struct {
	Config    *Config
	ircClient *twitch.Client
	apiClient *twitchapi.Connection
	commands  *cmd.Engine

	// This should eventually be private and hand out namespaces to modules.
	DB *storm.DB

	alert    *AlertService
	giveaway *GiveawayService
	marathon *MarathonService

	// For testing.  Unsure what the best way to handle this longterm.
	onConnect func()
	cmdErr    error
}

// CommandContext is passed to commands that are executed.
type CommandContext struct {
	Bot     *Bot
	Channel string
	User    *twitch.User
	Message *twitch.Message

	UserLevel int

	API *twitchapi.Connection
	IRC *twitch.Client
}

// NewBot creates a new, unconnected bot.
func NewBot(config *Config) *Bot {
	db, err := storm.Open("bot.db")
	if err != nil {
		log.Fatalf("can't open storm db: %v", err)
	}

	b := &Bot{
		Config:    config,
		DB:        db,
		ircClient: twitch.NewClient(config.BotUsername, "oauth:"+config.IRCOAuth),
		apiClient: twitchapi.NewConnection(config.ClientID, config.APIOAuth),
		commands:  cmd.NewEngine(),
	}

	b.AddCommand("game", "Tells the channel the current game.", gameCommand, 0)
	b.AddCommand("giveaway", "Giveaway", giveawayCommand, 0)
	b.AddCommand("marathon", "Show/Manipulate current maraton.", marathonCommand, 0)
	b.AddCommand("m", "alias for marathon.", marathonCommand, 0)
	b.AddCommand("setgame", "Sets the stream game.", setGameCommand, 10)

	if config.IRCAddress != "" {
		b.ircClient.IrcAddress = config.IRCAddress
	}
	if config.APIURLBase != "" {
		b.apiClient.UrlBase = config.APIURLBase
	}

	b.ircClient.OnConnect(b.handleConnect)
	b.ircClient.OnNewMessage(b.handleMessage)

	b.ircClient.Join(b.Config.Channel)

	return b
}

// AddCommand adds a bot command.
func (b *Bot) AddCommand(name string, help string,
	handler func(interface{}, []string) error,
	userLevel int) error {
	return b.commands.AddCommand(name, help, handler, userLevel)
}

// Connect the bot to Twitch.
func (b *Bot) Connect() error {
	b.startWebserver()
	return b.ircClient.Connect()
}

func (b *Bot) handleMessage(channel string, user twitch.User, message twitch.Message) {
	log.Println(channel)
	log.Println()
	if strings.HasPrefix(message.Text, "!") {
		ctx := &CommandContext{
			Bot:       b,
			Channel:   channel,
			User:      &user,
			UserLevel: b.userLevel(user.Username),
			Message:   &message,
			API:       b.apiClient,
			IRC:       b.ircClient,
		}
		b.cmdErr = b.commands.ExecString(
			ctx, b.userLevel(user.Username),
			strings.TrimPrefix(message.Text, "!"))
		if b.cmdErr != nil {
			log.Printf("Can't exec \"%s\": %v.", message.Text, b.cmdErr)
		}
	}
}

func (b *Bot) handleConnect() {
	if b.onConnect != nil {
		b.onConnect()
	}
}

func (b *Bot) userLevel(username string) int {
	log.Println(username)
	if username == b.Config.AdminUser {
		return 100
	} else {
		return 0
	}
}

func (b *Bot) isAdminRequest(r *http.Request) bool {
	// Internal Request
	if r == nil {
		return true
	}

	return r.Header.Get("Client-ID") == b.Config.ClientID &&
		r.Header.Get("Authorization") == ("OAuth "+b.Config.APIOAuth)
}
