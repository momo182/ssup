package appinit

import (
	"fmt"
	"os"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/dump"
	"github.com/momo182/ssup/src/entity"
)

// specialTargetMode is a mode where all args are Target names
// and networks are actually defined inside Supfile
func specialTargetMode(initData entity.InitState, helpMenu entity.HelpDisplayer) (*entity.PlayBook, error) {
	l := kemba.
		New("usecase::ParseInitialArgs::specialTargetMode").Printf

	conf := initData.Conf
	args := initData.InitialArgs.CommandArgs
	result := new(entity.PlayBook)

	// first lets find all targets defined by args
	for _, targetName := range args {
		l("targetName: %v", targetName)
		commands, ok := conf.Targets.Get(targetName)
		if !ok {
			fmt.Printf("ERR: 526BEDFF-F65F-4454-9B7C-32DA33F0DBD3, failed to get commands from target")
			os.Exit(41)
		}
		// having names of all commands, grab all parts associated with those
		for _, commandName := range commands {
			affix, ok := conf.Targets.GetAffixByCommandName(commandName)
			if !ok {
				fmt.Printf("ERR: C2E4EA9A-427E-47A5-A159-9E2DEA947903, failed to get affix from command")
				os.Exit(41)
			}
			netName := affix.AffixedNetwork
			l("affixedName: %v", affix)
			ensureNetworkExists(netName, conf, helpMenu)
			affixedNet, err := conf.GetNetworkByName(netName)
			if err != nil {
				return nil, err
			}

			// this should be common
			l("parse CLI --env flag env vars, override values defined in Network env")
			overrideEnvFromArgs(initData.InitialArgs.EnvVars, affixedNet)
			// this should be common
			addSSUPDefaultEnvs(affixedNet, args)

			command, ok := conf.Commands.Get(commandName)
			if !ok {
				fmt.Printf("ERR: C2E4EA9A-427E-47A5-A159-9E2DEA947903, failed to get command")
			}
			l("command: %v", command)
			l("network: %v", affixedNet)
			play := entity.Play{}
			play.Commands = append(play.Commands, &command)
			play.Network = affixedNet
			result.AddPlay(play)
		}
	}

	l("done affixes v1")
	l("dump: 96714924-ADA3-4F95-AFB2-21EEB37C5047")
	l("%s", dump.Format(result))

	return result, nil
}
