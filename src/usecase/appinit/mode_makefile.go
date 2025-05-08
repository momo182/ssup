package appinit

import (
	"fmt"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
	uc "github.com/momo182/ssup/src/usecase"
)

func makeFileMode(initData entity.InitState, helpMenu entity.HelpDisplayer) (*entity.PlayBook, error) {
	l := kemba.New("usecase::makeFileMode").Printf
	l("makefile mode selected")
	args := initData.InitialArgs.CommandArgs
	conf := initData.Conf
	result := new(entity.PlayBook)
	play := entity.Play{}

	uc.EnsureLocalhost(conf)
	network, err := conf.GetNetworkByName("localhost")
	if err != nil {
		return nil, err
	}
	network.Name = "localhost"
	play.Network = network

	// this should be common
	l("parse CLI --env flag env vars, override values defined in Network env")
	overrideEnvFromArgs(initData.InitialArgs.EnvVars, network)
	// this should be common
	addSSUPDefaultEnvs(network, args)

	for _, arg := range args {
		isCommand := conf.Commands.Has(arg)
		isTarget := conf.Targets.Has(arg)

		if isCommand {
			command, ok := conf.Commands.Get(arg)
			if !ok {
				return nil, fmt.Errorf("command %s not found", arg)
			}

			play.Commands = append(play.Commands, &command)
		}

		if isTarget {
			commandNames, ok := conf.Targets.Get(arg)
			if !ok {
				return nil, fmt.Errorf("command %s not found", arg)
			}

			for _, commandName := range commandNames {
				command, ok := conf.Commands.Get(commandName)
				if !ok {
					return nil, fmt.Errorf("command %s not found", commandName)
				}

				play.Commands = append(play.Commands, &command)
			}
		}

		result.AddPlay(play)
	}
	result.MarkAsMakefileMode()
	return result, nil
}
