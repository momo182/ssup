package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/clok/kemba"
	"github.com/davecgh/go-spew/spew"
	"github.com/gookit/goutil/dump"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/gateway/namespace"
	"github.com/momo182/ssup/src/gateway/shellcheck"
	svc "github.com/momo182/ssup/src/lobby"
	"github.com/momo182/ssup/src/usecase"
	oopslogrus "github.com/samber/oops/loggers/logrus"
	"github.com/sirupsen/logrus"
)

var RcloneConfig = ""

var initialArgs *entity.InitialArgs = &entity.InitialArgs{}

func init() {
	flag.StringVar(&initialArgs.Supfile, "f", "", "Custom path to ./Supfile[.yml]")
	flag.Var(&initialArgs.EnvVars, "e", "Set environment variables")
	flag.Var(&initialArgs.EnvVars, "env", "Set environment variables")
	flag.StringVar(&initialArgs.SshConfig, "sshconfig", "", "Read SSH Config file, ie. ~/.ssh/config file")
	flag.StringVar(&initialArgs.OnlyHosts, "only", "", "Filter hosts using regexp")
	flag.StringVar(&initialArgs.ExceptHosts, "except", "", "Filter out hosts using regexp")

	flag.BoolVar(&initialArgs.Debug, "D", false, "Enable debug mode")
	flag.BoolVar(&initialArgs.Debug, "debug", false, "Enable debug mode")
	flag.BoolVar(&initialArgs.DisablePrefix, "disable-prefix", false, "Disable hostname prefix")

	flag.BoolVar(&initialArgs.ShowVersion, "v", false, "Print version")
	flag.BoolVar(&initialArgs.ShowVersion, "version", false, "Print version")
	flag.BoolVar(&initialArgs.DisableColor, "c", false, "Disable color")
	flag.BoolVar(&initialArgs.DisableColor, "no-color", false, "Disable color")
	flag.BoolVar(&initialArgs.ShowHelp, "h", false, "Show help")
	flag.BoolVar(&initialArgs.ShowHelp, "help", false, "Show help")

	logrus.SetFormatter(oopslogrus.NewOopsFormatter(&logrus.JSONFormatter{
		PrettyPrint: true,
	}))

	spew.Config.MaxDepth = entity.SPEW_DEPTH
	svc.Lobby = &svc.ServiceLobby{}
	svc.Lobby.Shellcheck = &shellcheck.ShellCheck{}
	svc.Lobby.Namespaces = namespace.New()
}

func main() {
	l := kemba.New("main").Printf
	flag.Parse()
	initialArgs.CommandArgs = flag.Args()

	if initialArgs.ShowHelp {
		fmt.Fprintln(os.Stderr, entity.ErrUsage, "\n\nOptions:")
		flag.PrintDefaults()
		return
	}

	if initialArgs.ShowVersion {
		fmt.Fprintln(os.Stderr, entity.VERSION)
		return
	}
	l("reading supfile")
	conf := usecase.ReadSupfile(initialArgs)

	l("parse network and commands to be run from args")
	initData := entity.InitState{
		Conf:        conf,
		InitialArgs: initialArgs,
	}

	// MOD here we need to return not just filled single network
	// as it was before
	// in case of target with multiple affixes we
	// have to form playbook

	playbook, err := usecase.ParseInitialArgs(initData)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	switch {
	case playbook == nil:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	default:
		l("playbook is nil: negative checks passed")
		l("%v", dump.Format(playbook))
	}

	for _, play := range playbook.GetPlays() {
		switch {
		case play.Nets == nil:
			fmt.Fprintln(os.Stderr, fmt.Errorf("got nil pointer when parsing network"))
			os.Exit(3)
		default:
			l("play is nil: negative checks passed")
		}

		network := play.Nets
		commands := play.Commands
		l("starting play run\nhosts: %v\ncommands: %v\n", len(network.Hosts), len(commands))

		if e := usecase.RunShellcheck(conf); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1442)
		}

		usecase.CheckInitialArgs(network, initialArgs)
		vars := usecase.MergeVars(conf, network)

		l("parse CLI --env flag env vars") // define $SUP_ENV and override values defined in Supfile.
		usecase.SetEnvValues(&vars, initialArgs)
		usecase.GenerateSUPENVFrom(&vars)

		l("create new Stackup app")
		app, err := usecase.NewStackup(conf)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(11)
		}

		app.Debug(initialArgs.Debug)
		app.Prefix(!initialArgs.DisablePrefix)
		app.Args = initialArgs

		l("run all the commands in the given network")

		err = app.Run(network, vars, commands...)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(12)
		}
	}
}
