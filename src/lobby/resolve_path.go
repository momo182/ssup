package lobby

import (
	"os/user"
	"path/filepath"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/cliutil"
)

// ResolvePath resolves a path relative to the current working directory
func ResolvePath(path string) string {
	l := kemba.New("usecase::ResolvePath").Printf
	l("resolving given path: %s", path)
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
