package roll

import (
	"fmt"
	"log"
	"net/http"
)

type Giveaway struct {
	ID           int      `json:"id" storm:"id,increment"`
	Tag          string   `json:"tag"`
	Desc         string   `json:"desc"`
	Participants []string `json:"participants"`
}

type GiveawayService struct {
	bot *Bot
}

func NewGiveawayService(bot *Bot) *GiveawayService {
	return &GiveawayService{
		bot: bot,
	}
}

func (s *GiveawayService) New(r *http.Request, g *Giveaway, id *int) error {
	g.ID = 0
	return s.Update(r, g, id)
}

func (s *GiveawayService) Update(r *http.Request, g *Giveaway, id *int) error {
	if !s.bot.isAdminRequest(r) {
		return fmt.Errorf("access denied")
	}
	err := s.bot.DB.From("giveaway").Save(g)
	if err != nil {
		*id = -1
		return err
	}

	*id = g.ID
	return nil
}

func (s *GiveawayService) Get(r *http.Request, id *int, g *Giveaway) error {
	err := s.bot.DB.From("giveaway").One("ID", *id, g)
	log.Printf("%#v", g)
	return err
}

func giveawayDesc(cc *CommandContext) error {
	var giveaways []Giveaway
	err := cc.Bot.DB.From("giveaway").All(&giveaways)
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

func giveawayCommand(ctx interface{}, args []string) error {
	cc, ok := ctx.(*CommandContext)
	if !ok {
		return fmt.Errorf("ctx not a CommandContext")
	}

	if len(args) == 0 {
		return giveawayDesc(cc)
	}

	follows, err := cc.API.GetChannelFollows(cc.Channel)
	if err != nil {
		return err
	}
	isFollower := false
	for _, f := range follows.Follows {
		log.Printf("%#v", f)
		log.Printf("%d %d", int64(f.User.ID), cc.User.UserID)
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
	err = cc.Bot.DB.From("giveaway").All(&giveaways)
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
	err = cc.Bot.DB.From("giveaway").Save(giveaway)
	if err != nil {
		return err
	}

	cc.IRC.Say(cc.Channel, fmt.Sprintf("%s, you're now registered for the %s giveaway.",
		cc.User.Username, giveaway.Desc))
	return nil
}
