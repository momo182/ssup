package usecase

import (
	"os/exec"

	"github.com/clok/kemba"
	"github.com/pkg/errors"
)

// ResolveLocalPath - Use bash to resolve $ENV_VARs.
// like: `~/dir` or `$HOME/dir`
func ResolveLocalPath(cwd, path, env string) (string, error) {
	l := kemba.New("usecase::ResolveLocalPath").Printf
	// Check if file exists first.
	l("resolving variable: " + path)
	cmd := exec.Command("bash", "-c", env+"echo -n "+path)
	cmd.Dir = cwd
	resolvedFilename, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "resolving path failed")
	}

	return string(resolvedFilename), nil
}
