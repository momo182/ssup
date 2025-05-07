package appinit

import (
	"os"
	"os/exec"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
	"github.com/pkg/errors"
	"github.com/samber/oops"
)

func ResolveValues(e *entity.EnvList) error {
	l := kemba.New("usecase::ResolveValues").Printf
	if len(e.Keys()) == 0 {
		return nil
	}

	exports := ""
	for _, key := range e.Keys() {
		// inspect the value as a shell variable
		value := e.Get(key)
		l("looking for env: %v, var: %v", key, value)
		// check if value is prefixed with `$(`
		if entity.IsShell(value) {
			value, e := entity.ResolveShell(value)
			if e != nil {
				return oops.Trace("BC75A6AE-F1D8-4F15-A63D-D3A757E54481").
					Hint("resolving value via shell").
					With("value", value).
					Wrap(e)
			}
		}
		v := &entity.EnvVar{
			Key:   key,
			Value: value,
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

		e.Set(key, string(resolvedValue))
	}

	return nil
}
