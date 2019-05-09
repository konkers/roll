package roll

import (
	"fmt"

	"github.com/konkers/cmd"
)

type CmdEngine struct {
	*cmd.Engine
}

func NewCmdEngine() *CmdEngine {
	return &CmdEngine{
		Engine: cmd.NewEngine(),
	}
}

func (e *CmdEngine) AddCommand(name string, help string,
	handler func(*CommandContext, []string) error,
	userLevel int) error {
	proxyHandler := func(ctx interface{}, args []string) error {
		cc, ok := ctx.(*CommandContext)
		if !ok {
			return fmt.Errorf("ctx not a CommandContext")
		}
		return handler(cc, args)
	}
	return e.Engine.AddCommand(name, help, proxyHandler, userLevel)
}
