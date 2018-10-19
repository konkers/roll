package giveaway

import (
	"fmt"
	"net/http"
)

type SimpleCommandService struct {
	module *SimpleCommandModule
}

type SimpleCommandList struct {
	Commands []SimpleCommand `json :"commands"`
}

func NewSimpleCommandService(module *SimpleCommandModule) *SimpleCommandService {
	return &SimpleCommandService{
		module: module,
	}
}

func (s *SimpleCommandService) New(r *http.Request, g *SimpleCommand, id *int) error {
	g.ID = 0
	return s.Update(r, g, id)
}

func (s *SimpleCommandService) Update(r *http.Request, g *SimpleCommand, id *int) error {
	if !s.module.bot.IsAdminRequest(r) {
		return fmt.Errorf("access denied")
	}

	isNewCmd := g.ID == 0

	err := s.module.db.Save(g)
	if err != nil {
		*id = -1
		return err
	}

	*id = g.ID

	if isNewCmd {
		s.module.activateCommand(g)
	}
	return nil
}

func (s *SimpleCommandService) Get(r *http.Request, id *int, g *SimpleCommand) error {
	return s.module.db.One("ID", *id, g)
}

func (s *SimpleCommandService) Del(r *http.Request, id *int, ret *int) error {
	var cmd SimpleCommand
	err := s.module.db.One("ID", *id, &cmd)
	if err != nil {
		return err
	}

	s.module.deactivateCommand(&cmd)
	err = s.module.db.DeleteStruct(&cmd)
	if err != nil {
		return err
	}
	*ret = *id
	return nil
}

func (s *SimpleCommandService) All(r *http.Request, id *int, g *SimpleCommandList) error {
	return s.module.db.All(&g.Commands)
}
