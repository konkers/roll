package giveaway

import (
	"fmt"
	"log"

	"github.com/asdine/storm"
	"github.com/konkers/roll"
)

type Giveaway struct {
	ID           int      `json:"id" storm:"id,increment"`
	Tag          string   `json:"tag"`
	Desc         string   `json:"desc"`
	Participants []string `json:"participants"`
}

type GiveawayModule struct {
	bot *roll.Bot
	db  storm.Node

	service *GiveawayService
}

func init() {
	roll.RegisterModuleFactory(NewGiveawayModule, "giveaway")
}

func NewGiveawayModule(bot *roll.Bot, dbBucket storm.Node) (roll.Module, error) {
	module := &GiveawayModule{
		bot: bot,
		db:  dbBucket,
	}
	module.service = NewGiveawayService(module)

	bot.AddCommand("giveaway", "Giveaway command", module.giveawayCommand, 0)

	return module, nil
}

func (m *GiveawayModule) Start() error {
	return nil
}

func (m *GiveawayModule) Stop() error {
	return nil
}

func (m *GiveawayModule) GetRPCService() interface{} {
	return m.service
}

func (m *GiveawayModule) giveawayDesc(cc *roll.CommandContext) error {
	var giveaways []Giveaway
	err := m.db.All(&giveaways)
	if err != nil {
		return err
	}

	cc.IRC.Say(cc.Channel,
		"To register for one of the giveaways type !giveaway <tag>.  The list of tags are:")

	for _, g := range giveaways {
		cc.IRC.Say(cc.Channel, fmt.Sprintf("  %s - %s", g.Tag, g.Desc))
	}

	cc.IRC.Say(cc.Channel, "More information at: https://roll.konkers.net/")

	return nil
}

func (m *GiveawayModule) giveawayCommand(cc *roll.CommandContext, args []string) error {
	if len(args) == 0 {
		return m.giveawayDesc(cc)
	}

	follows, err := cc.API.GetChannelFollows(cc.Channel)
	if err != nil {
		return err
	}
	isFollower := false
	for _, f := range follows.Follows {
		if int64(f.User.ID) == cc.User.UserID {
			isFollower = true
			break
		}
	}

	if !isFollower {
		cc.IRC.Say(cc.Channel, "Giveaway only open to followers.  Please follow and try again :)")
		return nil
	}

	var giveaways []Giveaway
	err = m.db.All(&giveaways)
	if err != nil {
		return err
	}

	var giveaway *Giveaway
	for _, g := range giveaways {
		if g.Tag == args[0] {
			giveaway = &g
			break
		}
	}

	if giveaway == nil {
		cc.IRC.Say(cc.Channel,
			fmt.Sprintf("There's no %s giveaway.  Type !giveaway for a list", args[0]))
		return nil
	}

	log.Printf("%#v", giveaway)
	for _, p := range giveaway.Participants {
		if p == cc.User.Username {
			cc.IRC.Say(cc.Channel, fmt.Sprintf("%s, you're already registered.",
				cc.User.DisplayName))
			return nil
		}
	}

	giveaway.Participants = append(giveaway.Participants, cc.User.Username)
	err = m.db.Save(giveaway)
	if err != nil {
		return err
	}

	cc.IRC.Say(cc.Channel, fmt.Sprintf("%s, you're now registered for the %s giveaway.",
		cc.User.Username, giveaway.Desc))
	return nil
}
