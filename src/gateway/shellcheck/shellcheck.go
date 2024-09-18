package shellcheck

import (
	"fmt"
	"os/exec"

	"github.com/bitfield/script"
	"github.com/momo182/ssup/src/entity"
	"github.com/samber/oops"
)

type ShellCheck struct{}

func New() *ShellCheck {
	return &ShellCheck{}
}

func (s *ShellCheck) Check(cmd string) error {
	_, err := exec.LookPath("shellcheck")
	if err == nil {
		check := "shellcheck -f tty -e SC2148,SC2155,SC2001 -"
		fmt.Print(entity.ResetColor)
		_, e := script.Echo(cmd).Exec(check).Stdout()
		if e != nil {
			return oops.
				Trace("5386853C-E58D-4DBE-99F9-EE23C1E2444E").
				Hint("running shellcheck").
				Wrap(e)
		}
	}
	return nil
}
