package appinit

import (
	"fmt"
	"os"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/dump"
	"github.com/momo182/ssup/src/entity"
)

func normalMode(initData entity.InitState, helpMenu entity.HelpDisplayer) (*entity.PlayBook, error) {
	l := kemba.New("usecase::ParseInitialArgs::normalMode").Printf

	// defines
	result := new(entity.PlayBook)
	play := entity.Play{}
	envFromArgs := initData.InitialArgs.EnvVars
	args := initData.InitialArgs.CommandArgs
	conf := initData.Conf
	networkName := args[0]
	args = args[1:]

	// this will exit on err
	ensureNetworkExists(networkName, conf, helpMenu)
	network, err := conf.GetNetworkByName(networkName)
	if err != nil {
		return nil, err
	}

	l("parse CLI --env flag env vars, override values defined in Network env")
	overrideEnvFromArgs(envFromArgs, network)

	l("check if we have an inventory via script execution")
	hosts, err := network.ParseInventory()
	if err != nil {
		return nil, err
	}
	network.Hosts = append(network.Hosts, hosts...)

	addSSUPDefaultEnvs(network, initData.InitialArgs.CommandArgs)

	// actually add the network to the play
	play.Network = network

	for _, singleArgument := range args {
		target, isTarget := conf.Targets.Get(singleArgument)
		l("parse given command: %v", singleArgument)
		l("check if its a target")

		command, isCommand := conf.Commands.Get(singleArgument)
		if isCommand {
			l("found command: %v", singleArgument)
			play.Commands = append(play.Commands, &command)
		}

		if isTarget {
			l("found target: %v", singleArgument)
			for _, commandName := range target {
				targetCommand, ok := conf.Commands.Get(commandName)
				if !ok {
					fmt.Printf("ERR: 64B2D565-8345-4108-B790-25606C2128C0, command not found: %v, while traversing Targets:", commandName)
					os.Exit(1)
				}
				play.Commands = append(play.Commands, &targetCommand)
			}
		}

		l("check if both searches failed")
		if !isTarget && !isCommand {
			// cmdUsage(conf)
			helpMenu.ShowCmd = true
			helpMenu.Show(conf)
			return nil, fmt.Errorf("%v: %v", entity.ErrCmd, singleArgument)
		}
	}

	// add play into resulting playbook
	result.AddPlay(play)
	l("dump: A0ED3871-1622-4D93-BCBF-1924CE2828A9")
	l("%s", dump.Format(result))

	return result, nil
}
