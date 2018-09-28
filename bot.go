package roll

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/asdine/storm"
	twitch "github.com/gempir/go-twitch-irc"
	"github.com/konkers/twitchapi"
)

// Bot is the main object that represents the bot.
type Bot struct {
	Config    *Config
	ircClient *twitch.Client
	apiClient *twitchapi.Connection
	commands  *CmdEngine

	// This should eventually be private and hand out namespaces to modules.
	db *storm.DB

	modules map[string]Module

	funcMap template.FuncMap

	// For testing.  Unsure what the best way to handle this longterm.
	cmdErr error
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

// Module is the basic interface for bot modules.
type Module interface {
	Start() error
	Stop() error
}

// PublicWebProvider is implemented by modules that serve public web pages.
type PublicWebProvider interface {
	GetPublicHandler() http.Handler
}

// AdminWebProvider is implemented by modules that serve admin web pages.
type AdminWebProvider interface {
	GetAdminHandler() http.Handler
}

// RPCServiceProvider is implemented my modules that handle rpc requests.
type RPCServiceProvider interface {
	GetRPCService() interface{}
}

// ModuleFactory functions create modules.
type ModuleFactory func(bot *Bot, dbBucket storm.Node) (Module, error)

var moduleFactories = make(map[string]ModuleFactory)

func RegisterModuleFactory(f ModuleFactory, name string) error {
	if _, ok := moduleFactories[name]; ok {
		return fmt.Errorf("Module name \"%s\" registered more than once.", name)
	}
	moduleFactories[name] = f
	return nil
}

// NewBot creates a new, unconnected bot.
func NewBot(config *Config) (*Bot, error) {
	if config.DBPath == "" {
		config.DBPath = "bot.db"
	}
	db, err := storm.Open(config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("can't open storm db: %v", err)
	}

	b := &Bot{
		Config:    config,
		db:        db,
		modules:   make(map[string]Module),
		ircClient: twitch.NewClient(config.BotUsername, "oauth:"+config.IRCOAuth),
		apiClient: twitchapi.NewConnection(config.ClientID, config.APIOAuth),
		commands:  NewCmdEngine(),
	}

	if config.IRCAddress != "" {
		b.ircClient.IrcAddress = config.IRCAddress
	}
	if config.APIURLBase != "" {
		b.apiClient.UrlBase = config.APIURLBase
	}

	b.ircClient.OnNewMessage(b.handleMessage)

	b.ircClient.Join(b.Config.Channel)

	return b, nil
}

func (b *Bot) AddModule(modType string) error {
	return b.AddModuleByName(modType, modType)
}

func (b *Bot) AddModuleByName(modType string, name string) error {
	factory, ok := moduleFactories[modType]
	if !ok {
		return fmt.Errorf("Module type %s not found.", modType)
	}

	if _, ok = b.modules[name]; ok {
		return fmt.Errorf("Module named %s already registered.", name)
	}

	module, err := factory(b, b.db.From(name))
	if err != nil {
		return fmt.Errorf("Can't instantiate module %s: %v", modType, err)
	}

	b.modules[name] = module

	return nil
}

// AddCommand adds a bot command.
func (b *Bot) AddCommand(name string, help string,
	handler func(*CommandContext, []string) error,
	userLevel int) error {
	return b.commands.AddCommand(name, help, handler, userLevel)
}

// Connect the bot to Twitch.
func (b *Bot) Connect() error {
	// TODO: stop webserver on error
	err := b.startWebserver()
	if err != nil {
		return err
	}

	errChan := make(chan error)
	connectChan := make(chan bool)
	b.ircClient.OnConnect(func() {
		connectChan <- true
	})
	go func() {
		err := b.ircClient.Connect()
		errChan <- err
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("Can't connect to irc: %v", err)
		}
	case <-time.After(time.Second * 3):
		return fmt.Errorf("IRC connect timed out")
	case <-connectChan:
		// success case
	}

	for _, mod := range b.modules {
		mod.Start()
	}

	return nil
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

func (b *Bot) userLevel(username string) int {
	if username == b.Config.AdminUser {
		return 100
	} else {
		return 0
	}
}

func (b *Bot) IsAdminRequest(r *http.Request) bool {
	// Internal Request
	if r == nil {
		return true
	}

	return r.Header.Get("Client-ID") == b.Config.ClientID &&
		r.Header.Get("Authorization") == ("OAuth "+b.Config.APIOAuth)
}

func (b *Bot) Irc() *twitch.Client {
	return b.ircClient
}
