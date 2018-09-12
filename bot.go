package roll

import (
	"log"
	"strings"

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

	API *twitchapi.Connection
	IRC *twitch.Client
}

// NewBot creates a new, unconnected bot.
func NewBot(config *Config) *Bot {
	b := &Bot{
		Config:    config,
		ircClient: twitch.NewClient(config.BotUsername, config.IRCOAuth),
		apiClient: twitchapi.NewConnection(config.ClientID, config.APIOAuth),
		commands:  cmd.NewEngine(),
	}

	b.AddCommand("game", "Tells the channel the current game.", gameCommand, 0)
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
			Bot:     b,
			Channel: channel,
			User:    &user,
			Message: &message,
			API:     b.apiClient,
			IRC:     b.ircClient,
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
