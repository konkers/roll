package giveaway

import (
	"fmt"
	"net/http"
)

type GiveawayService struct {
	module *GiveawayModule
}

func NewGiveawayService(module *GiveawayModule) *GiveawayService {
	return &GiveawayService{
		module: module,
	}
}

func (s *GiveawayService) New(r *http.Request, g *Giveaway, id *int) error {
	g.ID = 0
	return s.Update(r, g, id)
}

func (s *GiveawayService) Update(r *http.Request, g *Giveaway, id *int) error {
	if !s.module.bot.IsAdminRequest(r) {
		return fmt.Errorf("access denied")
	}
	err := s.module.db.Save(g)
	if err != nil {
		*id = -1
		return err
	}

	*id = g.ID
	return nil
}

func (s *GiveawayService) Get(r *http.Request, id *int, g *Giveaway) error {
	err := s.module.db.One("ID", *id, g)
	return err
}
