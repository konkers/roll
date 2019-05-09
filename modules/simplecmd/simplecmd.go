package giveaway

import (
	"github.com/asdine/storm"
	"github.com/konkers/roll"
)

type SimpleCommand struct {
	ID       int    `json:"id" storm:"id,increment"`
	Command  string `json: "command"`
	Response string `json: "response"`
}

type SimpleCommandModule struct {
	bot *roll.Bot
	db  storm.Node

	service *SimpleCommandService
}

func init() {
	roll.RegisterModuleFactory(NewSimpleCommandModule, "simplecmd")
}

func NewSimpleCommandModule(bot *roll.Bot, dbBucket storm.Node) (roll.Module, error) {
	m := &SimpleCommandModule{
		bot: bot,
		db:  dbBucket,
	}
	m.service = NewSimpleCommandService(m)

	var cmds []SimpleCommand
	m.db.All(&cmds)
	for _, cmd := range cmds {
		m.activateCommand(&cmd)
	}

	return m, nil
}

func (m *SimpleCommandModule) Start() error {
	return nil
}

func (m *SimpleCommandModule) Stop() error {
	return nil
}

func (m *SimpleCommandModule) GetRPCService() interface{} {
	return m.service
}

func (m *SimpleCommandModule) activateCommand(cmd *SimpleCommand) {
	m.bot.AddCommand(cmd.Command, "Simple Command", func(cc *roll.CommandContext, args []string) error {
		return m.simpleCommand(cc, cmd.ID, args)
	}, 0)
}

func (m *SimpleCommandModule) deactivateCommand(cmd *SimpleCommand) {
	m.bot.RemoveCommand(cmd.Command)
}

func (m *SimpleCommandModule) simpleCommand(cc *roll.CommandContext, id int, args []string) error {
	var cmd SimpleCommand
	err := m.db.One("ID", id, &cmd)
	if err != nil {
		return err
	}
	cc.IRC.Say(cc.Channel, cmd.Response)
	return nil
}
