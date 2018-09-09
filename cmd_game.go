package roll

import (
	"fmt"
	"log"
	"strings"
)

func gameResponse(game string) string {
	return (fmt.Sprintf("The game is %s.", game))
}

func setGameResponse(game string) string {
	return (fmt.Sprintf("Game set to %s.", game))
}

func gameCommand(ctx interface{}, args []string) error {
	cc, ok := ctx.(*CommandContext)
	if !ok {
		return fmt.Errorf("ctx not a CommandContext")
	}

	channel, err := cc.API.GetChannel()
	if err != nil {
		return err
	}
	log.Printf("%#v", channel)

	cc.IRC.Say(cc.Channel, gameResponse(channel.Game))

	return nil
}

func setGameCommand(ctx interface{}, args []string) error {

	cc, ok := ctx.(*CommandContext)
	if !ok {
		return fmt.Errorf("ctx not a CommandContext")
	}

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
