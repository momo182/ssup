package appinit

import (
	"fmt"
	"log"
	"os"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/cliutil"
	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/strutil"
	"github.com/momo182/ssup/src/entity"
	uc "github.com/momo182/ssup/src/usecase"
	"gopkg.in/yaml.v2"
)

// ReadSupfile looks for Supfile or Supfiley.yml in the current working directory,
// cd's to Supfile dir, reads it and calls NewSupfile, after all returns the parsed Supfile.
func ReadSupfile(initialArgs *entity.InitialArgs) *entity.Supfile {
	l := kemba.New("usecase::read_supfile").Printf

	if initialArgs.Supfile == "" {
		l("no file specfied, assuming ./Supfile")
		initialArgs.Supfile = "./Supfile"
	}

	data, err := os.ReadFile(uc.ResolvePath(initialArgs.Supfile))
	if err != nil {
		firstErr := err
		l("failed to read ./Supfile, will try ./Supfile.yml")
		data, err = os.ReadFile("./Supfile.yml")
		if err != nil {
			l("failed to read ./Supfile.yml, will exit")
			fmt.Fprintln(os.Stderr, firstErr)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(47)
		}
	}
	l("successfully read ./Supfile")
	conf, err := NewSupfile(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(48)
	}

	// cd to supfile dir
	if initialArgs.Supfile != "" {
		l("cd to supfile dir: %s", initialArgs.Supfile)
		newWd := fsutil.Dir(initialArgs.Supfile)
		currName := fsutil.Name(initialArgs.Supfile)
		e := os.Chdir(newWd)
		if e != nil {
			log.Fatal("failed to cd to new Wd")
		}
		if !strutil.HasOneSub(string(initialArgs.Supfile[0]), []string{".", "/"}) {
			wd := cliutil.Workdir()
			initialArgs.Supfile = wd + "/" + currName
		}
		l("cd done")
	}

	l("successfully parsed Supfile")
	return conf
}

// NewSupfile parses configuration file and returns Supfile or error.
func NewSupfile(data []byte) (*entity.Supfile, error) {
	l := kemba.New("usecase > new_supfile").Printf
	var conf entity.Supfile
	l("parsing Supfile")

	if err := yaml.Unmarshal(data, &conf); err != nil {
		l("failed to parse Supfile, will exit")
		return nil, err
	}

	// API backward compatibility. Will be deprecated in v1.0.
	switch conf.Version {
	case "":
		conf.Version = "0.1"
		fallthrough

	case "0.1":
		for _, cmd := range conf.Commands.Cmds {
			if cmd.RunOnce {
				return nil, entity.ErrMustUpdate{Msg: "command.run_once is not supported in Supfile v" + conf.Version}
			}
		}
		fallthrough

	case "0.2":
		for _, cmd := range conf.Commands.Cmds {
			if cmd.Once {
				return nil, entity.ErrMustUpdate{Msg: "command.once is not supported in Supfile v" + conf.Version}
			}
			if cmd.Local != "" {
				return nil, entity.ErrMustUpdate{Msg: "command.local is not supported in Supfile v" + conf.Version}
			}
			if cmd.Serial != 0 {
				return nil, entity.ErrMustUpdate{Msg: "command.serial is not supported in Supfile v" + conf.Version}
			}
		}
		for _, network := range conf.Networks.Nets {
			if network.Inventory != "" {
				return nil, entity.ErrMustUpdate{Msg: "network.inventory is not supported in Supfile v" + conf.Version}
			}
		}
		fallthrough

	case "0.3":
		var warning string
		for key, cmd := range conf.Commands.Cmds {
			if cmd.RunOnce {
				warning = "Warning: command.run_once was deprecated by command.once in Supfile v" + conf.Version + "\n"
				cmd.Once = true
				conf.Commands.Cmds[key] = cmd
			}
		}
		if warning != "" {
			fmt.Fprintf(os.Stderr, warning)
		}

		fallthrough

	case "0.4", "0.5":

	default:
		return nil, entity.ErrUnsupportedSupfileVersion{Msg: "unsupported Supfile version " + conf.Version}
	}

	return &conf, nil
}
