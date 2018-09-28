package roll

import (
	"fmt"
	"strings"

	"github.com/asdine/storm"
	"github.com/konkers/roll"
)

type GameModule struct {
}

func NewGameModule(bot *roll.Bot, dbBucket storm.Node) (roll.Module, error) {
	module := &GameModule{}

	bot.AddCommand("game", "Lists the current game.", module.gameCommand, 0)
	bot.AddCommand("setgame", "Sets the current game.", module.setGameCommand, 10)

	return module, nil
}

func (m *GameModule) Start() error {
	return nil
}

func (m *GameModule) Stop() error {
	return nil
}

func gameResponse(game string) string {
	return (fmt.Sprintf("The game is %s.", game))
}

func setGameResponse(game string) string {
	return (fmt.Sprintf("Game set to %s.", game))
}

func (m *GameModule) gameCommand(cc *roll.CommandContext, args []string) error {
	channel, err := cc.API.GetChannel()
	if err != nil {
		return err
	}

	cc.IRC.Say(cc.Channel, gameResponse(channel.Game))
	return nil
}

func (m *GameModule) setGameCommand(cc *roll.CommandContext, args []string) error {
	channel, err := cc.API.GetChannel()
	if err != nil {
		return err
	}

	game := strings.Join(args, " ")

	err = cc.API.SetChannelGame(channel.Name, game)
	if err != nil {
		return err
	}
	cc.IRC.Say(cc.Channel, setGameResponse(game))

	return nil
}
