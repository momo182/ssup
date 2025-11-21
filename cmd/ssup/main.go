package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/clok/kemba"
	"github.com/goforj/godump"
	"github.com/momo182/ssup/src/entity"
	"github.com/samber/oops"

	// "github.com/momo182/ssup/src/gateway/namespace"
	"github.com/momo182/ssup/src/gateway/outrig"
	shellcheckService "github.com/momo182/ssup/src/gateway/shellcheck"
	"github.com/momo182/ssup/src/usecase"
	"github.com/momo182/ssup/src/usecase/appinit"
	_ "github.com/outrigdev/outrig/autoinit"
)

func init() {
	appinit.InitApp()
}

func main() {
	l := kemba.New("main").Printf

	// look what flags we have
	flag.Parse()
	entity.InitialArgsStore.CommandArgs = flag.Args()

	// tell outrig what we have
	outrig.Push("main_command_args", entity.InitialArgsStore)

	l("reading supfile")
	conf := appinit.ReadSupfile(entity.InitialArgsStore)
	if err := unpackInventories(conf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	appinit.ProcessSupfile(conf)

	// now if anyone wants some help
	if entity.InitialArgsStore.ShowHelp {
		fmt.Fprintln(os.Stderr, entity.ErrUsage, "\n\nOptions:")
		flag.PrintDefaults()
		return
	}

	if entity.InitialArgsStore.ShowExample {
		fmt.Println(entity.ExampleSupfile)
		return
	}

	if entity.InitialArgsStore.ShowVersion {
		fmt.Fprintln(os.Stderr, entity.VERSION)
		return
	}

	l("4685D4CC-E39C-49E7-A842-5ECEF446826A: supfile after inv process")
	l("%s", conf)
	outrig.Push("main_parsed_supfile", conf)

	l("parse network and commands to be run from args")
	initData := entity.InitState{
		Conf:        conf,
		InitialArgs: entity.InitialArgsStore,
	}

	// MOD here we need to return not just filled single network
	// as it was before
	// in case of target with multiple affixes we
	// have to form a playbook

	playbook, err := appinit.ParseInitialArgs(initData)

	// negative programming check
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if playbook == nil {
		fmt.Fprintln(os.Stderr, oops.Hint("4c14a210-c6bb-11f0-8e58-df6e80685e96: parsing playbook").Wrap(err))
		os.Exit(1)
	}

	l("playbook is not nil: negative checks passed")
	l("%v", godump.DumpStr(playbook))

	for _, play := range playbook.GetPlays() {
		isMakefileMode := playbook.IsMakefileMode()

		// more nagative programming checks
		if play.Network == nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("512ef020-c6bb-11f0-8e58-df6e80685e96: got nil pointer for network"))
			os.Exit(3)
		}
		l("play is not nil: negative checks passed")

		network := play.Network
		commands := play.Commands
		l("starting play run\nhosts: %v\ncommands: %v\n", len(network.Hosts), len(commands))

		if e := shellcheckService.RunShellcheck(conf); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(42)
		}

		usecase.CheckInitialArgs(network, entity.InitialArgsStore)
		vars := appinit.MergeVars(conf, network)

		l("parse CLI --env flag env vars") // define $SUP_ENV and override values defined in Supfile.
		appinit.SetEnvValues(&vars, entity.InitialArgsStore)
		appinit.GenerateSUPENVFrom(&vars)

		l("create new Stackup app")
		app := usecase.NewStackup(conf)

		app.Debug(entity.InitialArgsStore.Debug)
		app.Prefix(!entity.InitialArgsStore.DisablePrefix)
		app.Args = entity.InitialArgsStore

		l("run all the commands in the given network")
		runOrder := entity.RunOrder{
			MakefileMode: isMakefileMode,
			Network:      network,
			Vars:         vars,
			Commands:     commands,
			Args:         *entity.InitialArgsStore,
		}

		err = app.Run(runOrder)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} // end of for _, play := range playbook.GetPlays()

	// println("press ctrl+c")
	// select {}
}

func unpackInventories(conf *entity.Supfile) error {
	l := kemba.New("unpackInventories").Printf
	l("check if we have an inventory via script execution")
	result := make(map[string]entity.Network)

	for name, network := range conf.Networks.Nets {
		if network.Inventory != "" {
			hosts, err := network.ParseInventory()
			if err != nil {
				return err
			}
			network.Hosts = hosts
		}
		result[name] = network
	}

	outrig.Push("unpackInventories", result)
	conf.Networks.Nets = result

	return nil
}
