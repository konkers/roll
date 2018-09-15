package roll

import (
	"fmt"
	"net/http"
	"time"

	"github.com/konkers/cmd"
)

type MarathonGameStatus int

const (
	GameStatusNotStarted = MarathonGameStatus(iota)
	GameStatusRunning
	GameStatusFinished
)

type MarathonGame struct {
	Name        *string             `json:"name"`
	TwitchGame  *string             `json:"twitch_game"`
	Link        *string             `json:"link"`
	System      *string             `json:"system"`
	Status      *MarathonGameStatus `json:"status"`
	StartedTime *time.Time          `json:"started_time"`
	EndedTime   *time.Time          `json:"ended_time"`
}

type Marathon struct {
	ID    int             `json:"id" storm:"id,increment"`
	Name  *string         `json:"name"`
	Games []*MarathonGame `json:"games"`
}

type MarathonService struct {
	bot *Bot
}

const CurrentMarathon = 1

var marathonCmd = cmd.NewEngine()

func NewMarathonService(bot *Bot) *MarathonService {
	marathonCmd.AddCommand("next", "go to next game", marathonNextCommand, 10)
	marathonCmd.AddCommand("resetgame", "reset the game", marathonResetGameCommand, 10)
	marathonCmd.AddCommand("resetmarathon", "reset the marathon", marathonResetMarathonCommand, 10)
	return &MarathonService{
		bot: bot,
	}
}

func (s *MarathonService) New(r *http.Request, args *Marathon, reply *int) error {
	if !s.bot.isAdminRequest(r) {
		return fmt.Errorf("access denied")
	}
	args.ID = 0
	err := s.bot.DB.From("Marathon").Save(args)
	if err != nil {
		*reply = -1
		return err
	}

	*reply = args.ID
	return nil
}

func (s *MarathonService) Update(r *http.Request, args *Marathon, reply *int) error {
	if !s.bot.isAdminRequest(r) {
		return fmt.Errorf("access denied")
	}
	err := s.bot.DB.From("Marathon").Save(args)
	if err != nil {
		*reply = -1
		return err
	}

	*reply = args.ID
	return nil
}

func (s *MarathonService) Get(r *http.Request, id *int, marathon *Marathon) error {
	return s.bot.DB.From("Marathon").One("ID", *id, marathon)
}

func showMarathon(cc *CommandContext) error {
	var marathon Marathon
	cur := CurrentMarathon
	err := cc.Bot.marathon.Get(nil, &cur, &marathon)
	if err != nil {
		return err
	}
	game := marathon.CurrentGame()
	if game == nil {
		cc.IRC.Say(cc.Channel, fmt.Sprintf("Marathon is not running"))
	} else {
		cc.IRC.Say(cc.Channel, fmt.Sprintf("Current game is: %s", *game.Name))
	}
	return nil
}

func marathonCommand(ctx interface{}, args []string) error {
	cc, ok := ctx.(*CommandContext)
	if !ok {
		return fmt.Errorf("ctx not a CommandContext")
	}

	if len(args) == 0 {
		return showMarathon(cc)
	}

	return marathonCmd.Exec(cc, cc.UserLevel, args)
}

func marathonResetGameCommand(ctx interface{}, args []string) error {
	cc, ok := ctx.(*CommandContext)
	if !ok {
		return fmt.Errorf("ctx not a CommandContext")
	}
	var marathon Marathon
	cur := CurrentMarathon
	err := cc.Bot.marathon.Get(nil, &cur, &marathon)
	if err != nil {
		return err
	}

	marathon.ResetGame()
	reply := 0
	err = cc.Bot.marathon.Update(nil, &marathon, &reply)
	if err != nil {
		return err
	}
	cc.IRC.Say(cc.Channel, "RESET!")
	return nil
}

func marathonResetMarathonCommand(ctx interface{}, args []string) error {
	cc, ok := ctx.(*CommandContext)
	if !ok {
		return fmt.Errorf("ctx not a CommandContext")
	}
	var marathon Marathon
	cur := CurrentMarathon
	err := cc.Bot.marathon.Get(nil, &cur, &marathon)
	if err != nil {
		return err
	}

	marathon.ResetMarathon()
	reply := 0
	err = cc.Bot.marathon.Update(nil, &marathon, &reply)
	if err != nil {
		return err
	}
	cc.IRC.Say(cc.Channel, "RESET!")
	return nil
}

func marathonNextCommand(ctx interface{}, args []string) error {
	cc, ok := ctx.(*CommandContext)
	if !ok {
		return fmt.Errorf("ctx not a CommandContext")
	}
	var marathon Marathon
	cur := CurrentMarathon
	err := cc.Bot.marathon.Get(nil, &cur, &marathon)
	if err != nil {
		return err
	}

	prevGame := marathon.CurrentGame()
	marathon.NextGame()
	nextGame := marathon.CurrentGame()
	reply := 0
	err = cc.Bot.marathon.Update(nil, &marathon, &reply)
	if err != nil {
		return err
	}

	if prevGame != nil {
		cc.IRC.Say(cc.Channel, fmt.Sprintf("%s complete!", *prevGame.Name))
	}
	if nextGame != nil {
		cc.IRC.Say(cc.Channel, fmt.Sprintf("%s started!", *nextGame.Name))
		if nextGame.TwitchGame != nil {
			setGameCommand(ctx, []string{*nextGame.TwitchGame})
		} else {
			setGameCommand(ctx, []string{*nextGame.Name})
		}
	}
	return nil
}

func (m *Marathon) CurrentGame() *MarathonGame {
	for _, game := range m.Games {
		if game.Status != nil && *game.Status == GameStatusRunning {
			return game
		}
	}
	return nil
}

func (m *Marathon) NextGame() error {
	if len(m.Games) == 0 {
		return fmt.Errorf("No games.")
	}

	for _, game := range m.Games {
		if game.Status == nil {
			var status = GameStatusNotStarted
			game.Status = &status
		}

		if *game.Status == GameStatusNotStarted {
			*game.Status = GameStatusRunning
			t := time.Now()
			game.StartedTime = &t
			return nil

		} else if *game.Status == GameStatusRunning {
			*game.Status = GameStatusFinished
			t := time.Now()
			game.EndedTime = &t
		}
	}
	return nil
}

func (m *Marathon) ResetGame() error {
	for _, game := range m.Games {
		if game.Status != nil && *game.Status == GameStatusRunning {
			var status = GameStatusNotStarted
			game.Status = &status
			game.StartedTime = nil
			game.EndedTime = nil
		}
	}
	return nil
}

func (m *Marathon) ResetMarathon() error {
	for _, game := range m.Games {
		var status = GameStatusNotStarted
		game.Status = &status
		game.StartedTime = nil
		game.EndedTime = nil
	}
	return nil
}
