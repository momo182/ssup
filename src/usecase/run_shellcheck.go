package usecase

import (
	"github.com/momo182/ssup/src/entity"
	svc "github.com/momo182/ssup/src/lobby"
	"github.com/samber/oops"
)

func RunShellcheck(supFile *entity.Supfile) error {
	var errors []error = make([]error, 0)
	if supFile == nil {
		return oops.Trace("0FEB1489-CC42-43CC-8C6F-97ED76513004").
			Hint("supfile is nil").
			Errorf("supfile is nil")
	}

	// get all tasks from supfile
	tasks := supFile.Commands.Names

	for _, task := range tasks {
		command := supFile.Commands.Cmds[task]
		switch {
		case len(command.Run) > 0:
			if e := svc.Lobby.Shellcheck.Check(command.Run); e != nil {
				errors = append(errors, e)
			}
		case len(command.Script) > 0:
			if e := svc.Lobby.Shellcheck.Check(command.Script); e != nil {
				errors = append(errors, e)
			}
		case len(command.Local) > 0:
			if e := svc.Lobby.Shellcheck.Check(command.Local); e != nil {
				errors = append(errors, e)
			}
		}
	}

	if len(errors) > 0 {
		return oops.Trace("60030684-F634-461C-9657-98025D99B950").
			Hint("Shellcheck found errors in the supfile").
			Wrap(errors[len(errors)-1])
	}

	return nil
}
