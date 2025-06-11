package shellcheck

import (
	"strings"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
	"github.com/samber/oops"
)

func RunShellcheck(supFile *entity.Supfile) error {
	l := kemba.New("uc::RunShellcheck").Printf
	l("will run shellcheck on Supfile")
	var shellcheck entity.ShellCheckFacade
	shellcheck = &ShellCheckProvider{}
	var errors []error = make([]error, 0)
	if supFile == nil {
		return oops.Trace("0FEB1489-CC42-43CC-8C6F-97ED76513004").
			Hint("supfile is nil").
			Errorf("supfile is nil")
	}
	l("supfile is not nil: negative checks passed")

	// get all commandNames from supfile
	commandNames := supFile.Commands.Names

	// TODO this probably was broken when
	// i added plays support
	for _, commandName := range commandNames {
		l("inspected command: " + commandName)
		command := supFile.Commands.Cmds[commandName]
		command.Run = strings.TrimSpace(command.Run)

		l("len command.Run: %v", len(command.Run))
		l("len command.Script: %v", len(command.Script))
		l("len command.Local: %v", len(command.Local))
		if len(command.Run) > 0 {
			l("will run shellcheck on command: " + command.Run)

			firstLine := strings.Split(command.Run, "\n")[0]
			l("firstLine: %v", firstLine)

			// expect that #! in the first line is meant to override the interpreter
			// treat as the user knows what they are doing
			// and we do not want to stand in the way of that
			if strings.Contains(firstLine, "#!") {
				l("command.Run contains nosc tag, skipping shellcheck")
				continue
			}

			if e := shellcheck.Check(command.Run, commandName); e != nil {
				errors = append(errors, e)
			}
		} else {
			l("command.Run is empty, skipping shellcheck")
		}

		if len(command.Script) > 0 {
			l("will run shellcheck on script: " + command.Script)

			firstLine := strings.Split(command.Script, "\n")[0]
			if strings.Contains(firstLine, "fish") {
				l("command.Script contains fish, skipping shellcheck")
				continue
			}
			if strings.Contains(firstLine, "nu") {
				l("command.Script contains nu shell, skipping shellcheck")
				continue
			}

			if e := shellcheck.Check(command.Script, commandName); e != nil {
				errors = append(errors, e)
			}
		} else {
			l("command.Script is empty, skipping shellcheck")
		}

		if len(command.Local) > 0 {
			l("will run shellcheck on local: " + command.Local)

			firstLine := strings.Split(command.Local, "\n")[0]
			if strings.Contains(firstLine, "fish") {
				l("command.Local contains fish, skipping shellcheck")
				continue
			}
			if strings.Contains(firstLine, "nu") {
				l("command.Local contains nu shell, skipping shellcheck")
				continue
			}

			if e := shellcheck.Check(command.Local, commandName); e != nil {
				errors = append(errors, e)
			}
		} else {
			l("command.Local is empty, skipping shellcheck")
		}
	}

	if len(errors) > 0 {
		return oops.Trace("60030684-F634-461C-9657-98025D99B950").
			Hint("Shellcheck found errors in the supfile").
			Wrap(errors[len(errors)-1])
	}

	return nil
}
