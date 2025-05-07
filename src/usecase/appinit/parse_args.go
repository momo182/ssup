package appinit

import (
	"fmt"
	"os"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/lobby"

	// "github.com/momo182/ssup/src/usecase/modes"
	"github.com/no-src/nsgo/osutil"
)

// ParseInitialArgs parses args and returns network and commands to be run.
// On error, it prints usage and exits.
func ParseInitialArgs(initData entity.InitState) (*entity.PlayBook, error) {
	l := kemba.New("usecase::ParseInitialArgs").Printf

	conf := initData.Conf
	initialArgs := initData.InitialArgs
	args := initialArgs.CommandArgs
	argsCount := len(args)
	hasNetworks := false
	helpMenu := newHelpDisplayer(initialArgs)

	l("check we have any args at all, len: %v", argsCount)
	switch {
	case argsCount < 1: // no args given
		helpMenu.ShowAll(conf)
		return nil, entity.ErrUsage
	}

	netCount := len(conf.Networks.Names)
	l("check if Supfile has any networks defined: %v", netCount)
	if netCount > 0 {
		l("found networks: %v", netCount)
		hasNetworks = true
	}

	switch hasNetworks {
	case true:
		onlyTargets := allArgsAreTargets(initData, helpMenu)
		if onlyTargets {
			l("special target mode")
			makefileFlowPlayBook, err := SpecialTargetMode(initData, helpMenu)
			if err != nil {
				return nil, err
			}
			return makefileFlowPlayBook, nil
		}

		l("normal mode")
		normalFlowPlayBook, err := NormalMode(initData, helpMenu)
		if err != nil {
			return nil, err
		}
		return normalFlowPlayBook, nil
	default:
		l("makefile mode")
		makefileFlow, err := MakeFileMode(initData, helpMenu)
		if err != nil {
			return nil, err
		}
		return makefileFlow, nil
	}
}

func preparePlaybookForTarget(cmd string, conf *entity.Supfile, helpMenu entity.HelpDisplayer) (*entity.PlayBook, error) {
	l := kemba.New("usecase::ParseInitialArgs::preparePlaybookForTargets").Printf

	// negative programing checks
	switch {
	case conf == nil:
		fmt.Printf("ERR: 6A246CE5-12A5-4401-9B63-E92A8F1C6F51, conf is nil")
		os.Exit(1)
	}

	play := entity.Play{}
	result := new(entity.PlayBook)
	var commands []*entity.Command

	l("will try to append command to commands set")
	commands = ifExistsAppendTo(commands, cmd, conf, helpMenu)

	l("check if affixes present")
	if conf.Targets.HasAffixes() {
		l("found affixes")

		affix, ok := conf.Targets.GetAffixByCommandName(cmd)
		if !ok {
			helpMenu.ShowCmd = true
			helpMenu.Show(conf)
			fmt.Printf("Affix: '%s' does not exist", affix)
			os.Exit(24)
		}

		l("%v", affix)
		networkToCheck := affix.AffixedNetwork
		lobby.EnsureNetworkExists(networkToCheck, conf, helpMenu)
		affixedNet, err := conf.GetNetworkByName(networkToCheck)
		if err != nil {
			return nil, err
		}

		play.Commands = commands
		play.Network = affixedNet

		result.AddPlay(play)
	}
	return result, nil
}

func ifExistsAppendTo(commands []*entity.Command, cmd string, conf *entity.Supfile, helpMenu entity.HelpDisplayer) []*entity.Command {
	switch {
	case commands == nil:
		fmt.Printf("ERR: B53A3D5C-4972-42B7-8961-CFAB83ECD1B2, got nil command")
		os.Exit(41)
	case conf == nil:
		fmt.Printf("ERR: 3A70B9C1-9118-4B29-AB2B-2C4F3E864465, got nil conf")
		os.Exit(42)
	}

	command, isCommand := conf.Commands.Get(cmd)
	if !isCommand {
		helpMenu.ShowCmd = true
		helpMenu.Show(conf)
		fmt.Printf("Command: '%s' does not exist", cmd)
		os.Exit(24)
	}
	command.Name = cmd
	commands = append(commands, &command)
	return commands
}

func newHelpDisplayer(initialArgs *entity.InitialArgs) entity.HelpDisplayer {
	helpMenu := entity.HelpDisplayer{}
	helpMenu.Color = true

	// dont trust windows on colors, nushell over ssh played bad
	// don't expect this will be better
	if osutil.IsWindows() {
		helpMenu.Color = false
	}

	if initialArgs.DisableColor {
		helpMenu.Color = false
	}
	return helpMenu
}

func allArgsAreTargets(initData entity.InitState, helpMenu entity.HelpDisplayer) bool {
	l := kemba.New("usecase::ParseInitialArgs::allArgsAreTargets").Printf
	conf := initData.Conf
	args := initData.InitialArgs.CommandArgs
	noMissingNames := true
	l("check if all given args are targets: %v", len(args))

	for _, singleArgument := range args {
		if !conf.Targets.Has(singleArgument) {
			l("targets check -> unknown keyword: %v", singleArgument)
			noMissingNames = false
		}
	}

	if noMissingNames {
		return true
	}
	return false
}

func TargetsHaveAffixes(conf *entity.Supfile) bool {
	for _, selectedTarget := range conf.Targets.Names {
		if _, ok := conf.Targets.GetAffixByCommandName(selectedTarget); ok {
			return true
		}
	}
	return false
}
