package usecase

import (
	"os"
	"os/exec"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/shared/shellresolve"
	"github.com/pkg/errors"
	"github.com/samber/oops"
)

func ResolveValues(e *entity.EnvList) error {
	l := kemba.New("usecase > ResolveValues").Printf
	if len(*e) == 0 {
		return nil
	}

	exports := ""
	for i, v := range *e {
		// inspect the value as a shell variable
		value := v.Value
		l("looking for env: %v, var: %v", v.Key, value)
		// check if value is prefixed with `$(`
		if shellresolve.IsShell(value) {
			value, e := shellresolve.ResolveShell(value)
			if e != nil {
				return oops.Trace("BC75A6AE-F1D8-4F15-A63D-D3A757E54481").
					Hint("resolving value via shell").
					With("value", value).
					Wrap(e)
			}
		}

		exports += v.AsExport()

		cmd := exec.Command("bash", "-c", exports+"echo -n "+v.Value+";")
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		cmd.Dir = cwd
		resolvedValue, err := cmd.Output()
		if err != nil {
			return errors.Wrapf(err, "resolving env var %v failed", v.Key)
		}

		(*e)[i].Value = string(resolvedValue)
	}

	return nil
}
