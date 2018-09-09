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
	config    *Config
	ircClient *twitch.Client
	apiClient *twitchapi.Connection
	commands  *cmd.Engine

	// For testing.  Unsure what the best way to handle this longterm.
	onConnect func()
	cmdErr    error
}

// Config is the bot's configuration
type Config struct {
	BotUsername string `json:"bot_username"`
	Channel     string `json:"channel"`
	ClientID    string `json:"client_id"`
	APIOAuth    string `json:"api_oauth"`
	IRCOAuth    string `json:"irc_oauth"`
	IRCAddress  string `json:"irc_addr"`
	AdminUser   string `json:"admin_user"`
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
		config:    config,
		ircClient: twitch.NewClient(config.BotUsername, config.IRCOAuth),
		apiClient: twitchapi.NewConnection(config.ClientID, config.APIOAuth),
		commands:  cmd.NewEngine(),
	}

	b.AddCommand("game", "Tells the channel the current game.", gameCommand, 0)
	b.AddCommand("setgame", "Sets the stream game.", setGameCommand, 10)

	if config.IRCAddress != "" {
		b.ircClient.IrcAddress = config.IRCAddress
	}

	b.ircClient.OnConnect(b.handleConnect)
	b.ircClient.OnNewMessage(b.handleMessage)

	b.ircClient.Join(b.config.Channel)

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
	if username == b.config.AdminUser {
		return 100
	} else {
		return 0
	}
}
