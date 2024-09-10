package usecase

import (
	"os/user"
	"path/filepath"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/cliutil"
)

func ResolvePath(path string) string {
	l := kemba.New("usecase > ResolvePath").Printf
	l("ressolve path: %s", path)
	if path == "" {
		return ""
	}

	if path == "." {
		return cliutil.Workdir()
	}

	if path[:2] == "~/" {
		usr, err := user.Current()
		if err == nil {
			path = filepath.Join(usr.HomeDir, path[2:])
		}
	}
	l("final path: %s", path)
	return path
}
