package marathon

import (
	"fmt"
	"net/http"
	"time"

	"github.com/asdine/storm"
	"github.com/konkers/roll"
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

type MarathonModule struct {
	bot *roll.Bot
	db  storm.Node

	marathonCmd *roll.CmdEngine
	service     *MarathonService
}

type MarathonService struct {
	module *MarathonModule
}

func init() {
	roll.RegisterModuleFactory(NewMarathonModule, "marathon")
}

func NewMarathonModule(bot *roll.Bot, db storm.Node) (roll.Module, error) {
	module := &MarathonModule{
		bot:         bot,
		db:          db,
		marathonCmd: roll.NewCmdEngine(),
	}

	module.service = NewMarathonService(module)

	module.marathonCmd.AddCommand("next", "go to next game", module.marathonNextCommand, 10)
	module.marathonCmd.AddCommand("resetgame", "reset the game", module.marathonResetGameCommand, 10)
	module.marathonCmd.AddCommand("resetmarathon", "reset the marathon", module.marathonResetMarathonCommand, 10)

	if err := bot.AddTemplateFunc("status", renderStatus); err != nil {
		return nil, err
	}

	if err := bot.AddTemplateFunc("time", renderTime); err != nil {
		return nil, err
	}

	return module, nil
}

func (m *MarathonModule) Start() error {
	return nil
}

func (m *MarathonModule) Stop() error {
	return nil
}

const CurrentMarathon = 1

func NewMarathonService(module *MarathonModule) *MarathonService {
	return &MarathonService{
		module: module,
	}
}

func (s *MarathonService) New(r *http.Request, args *Marathon, reply *int) error {
	if !s.module.bot.IsAdminRequest(r) {
		return fmt.Errorf("access denied")
	}
	args.ID = 0
	err := s.module.db.Save(args)
	if err != nil {
		*reply = -1
		return err
	}

	*reply = args.ID
	return nil
}

func (s *MarathonService) Update(r *http.Request, args *Marathon, reply *int) error {
	if !s.module.bot.IsAdminRequest(r) {
		return fmt.Errorf("access denied")
	}
	err := s.module.db.Save(args)
	if err != nil {
		*reply = -1
		return err
	}

	*reply = args.ID
	return nil
}

func renderStatus(status *MarathonGameStatus) string {
	if status == nil {
		return "not started"
	}
	switch *status {
	case GameStatusNotStarted:
		return "not started"
	case GameStatusRunning:
		return "running"
	case GameStatusFinished:
		return "finished"
	default:
		return "???"
	}
}

func renderTime(game *MarathonGame) string {
	var d time.Duration
	if game.StartedTime == nil {
		return "?:??:??"
	} else if game.EndedTime == nil {
		d = time.Now().Sub(*game.StartedTime)
	} else {
		d = game.EndedTime.Sub(*game.StartedTime)
	}
	hours := d.Truncate(time.Hour)
	d -= hours
	mins := d.Truncate(time.Minute)
	d -= mins
	seconds := d.Truncate(time.Second)
	return fmt.Sprintf("%01d:%02d:%02d", int(hours.Hours()), int(mins.Minutes()), int(seconds.Seconds()))
}

func (s *MarathonService) Get(r *http.Request, id *int, marathon *Marathon) error {
	return s.module.db.One("ID", *id, marathon)
}

func (m *MarathonModule) showMarathon(cc *roll.CommandContext) error {
	var marathon Marathon
	cur := CurrentMarathon
	err := m.service.Get(nil, &cur, &marathon)
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

func (m *MarathonModule) marathonCommand(cc *roll.CommandContext, args []string) error {
	if len(args) == 0 {
		return m.showMarathon(cc)
	}

	return m.marathonCmd.Exec(cc, cc.UserLevel, args)
}

func (m *MarathonModule) marathonResetGameCommand(cc *roll.CommandContext, args []string) error {
	var marathon Marathon
	cur := CurrentMarathon
	err := m.service.Get(nil, &cur, &marathon)
	if err != nil {
		return err
	}

	marathon.ResetGame()
	reply := 0
	err = m.service.Update(nil, &marathon, &reply)
	if err != nil {
		return err
	}
	cc.IRC.Say(cc.Channel, "RESET!")
	return nil
}

func (m *MarathonModule) marathonResetMarathonCommand(cc *roll.CommandContext, args []string) error {
	var marathon Marathon
	cur := CurrentMarathon
	err := m.service.Get(nil, &cur, &marathon)
	if err != nil {
		return err
	}

	marathon.ResetMarathon()
	reply := 0
	err = m.service.Update(nil, &marathon, &reply)
	if err != nil {
		return err
	}
	cc.IRC.Say(cc.Channel, "RESET!")
	return nil
}

func (m *MarathonModule) marathonNextCommand(cc *roll.CommandContext, args []string) error {
	var marathon Marathon
	cur := CurrentMarathon
	err := m.service.Get(nil, &cur, &marathon)
	if err != nil {
		return err
	}

	prevGame := marathon.CurrentGame()
	marathon.NextGame()
	nextGame := marathon.CurrentGame()
	reply := 0
	err = m.service.Update(nil, &marathon, &reply)
	if err != nil {
		return err
	}

	if prevGame != nil {
		cc.IRC.Say(cc.Channel, fmt.Sprintf("%s complete!", *prevGame.Name))
	}
	if nextGame != nil {
		cc.IRC.Say(cc.Channel, fmt.Sprintf("%s started!", *nextGame.Name))
		if nextGame.TwitchGame != nil {
			cc.API.SetChannelGame(cc.Channel, *nextGame.TwitchGame)
		} else {
			cc.API.SetChannelGame(cc.Channel, *nextGame.Name)
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
