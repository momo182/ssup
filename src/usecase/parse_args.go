package usecase

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
)

// ParseInitialArgs parses args and returns network and commands to be run.
// On error, it prints usage and exits.
func ParseInitialArgs(conf *entity.Supfile, envFromArgs entity.FlagStringSlice) (*entity.Network, []*entity.Command, error) {
	l := kemba.New("usecase::ParseInitialArgs").Printf
	var commands []*entity.Command

	l("check args len")
	args := flag.Args()

	switch {
	// no args given
	case len(args) < 1:
		networkUsage(conf)
		cmdUsage(conf)
		return nil, nil, entity.ErrUsage

	case len(args) >= 1:
		l("check if all args are commands")
		if len(conf.Networks.Names) == 0 {
			l("no name of network")
			l("test that all other args are commands or targets")
			for _, cmd := range args[1:] {
				if !conf.Targets.Has(cmd) && !conf.Commands.Has(cmd) {
					l("tested both Commands and Targets")
					l("unknown command: %v", cmd)
					networkUsage(conf)
					cmdUsage(conf)
					return nil, nil, fmt.Errorf("%v: %v", entity.ErrCmd, cmd)
				}
			}

			EnsureLocalhost(conf)
			args = append([]string{"localhost"}, args...)
		}
	}

	l("does the <network> exist?")
	network, ok := conf.Networks.Get(args[0])
	if !ok {
		networkUsage(conf)
		return nil, nil, entity.ErrUnknownNetwork
	}

	l("parse CLI --env flag env vars, override values defined in Network env")
	for _, env := range envFromArgs {
		if len(env) == 0 {
			continue
		}
		i := strings.Index(env, "=")
		if i < 0 {
			if len(env) > 0 {
				network.Env.Set(env, "")
			}
			continue
		}
		network.Env.Set(env[:i], env[i+1:])
	}

	l("parse inventory")
	hosts, err := network.ParseInventory()
	if err != nil {
		return nil, nil, err
	}
	network.Hosts = append(network.Hosts, hosts...)

	l("does the <network> have at least one host?")
	if len(network.Hosts) == 0 {
		networkUsage(conf)
		return nil, nil, entity.ErrNetworkNoHosts
	}

	l("check for the second argument")
	if len(args) < 2 {
		cmdUsage(conf)
		return nil, nil, entity.ErrUsage
	}

	// l("in case of the network.Env needs an initialization")
	// if network.Env == nil {
	// 	network.Env = entity.EnvList{}
	// }

	l("add default env variable with current network")
	network.Env.Set("SUP_NETWORK", args[0])

	l("add default nonce")
	network.Env.Set("SUP_TIME", time.Now().UTC().Format(time.RFC3339))
	if os.Getenv("SUP_TIME") != "" {
		network.Env.Set("SUP_TIME", os.Getenv("SUP_TIME"))
	}

	l("add user")
	if os.Getenv("SUP_USER") != "" {
		network.Env.Set("SUP_USER", os.Getenv("SUP_USER"))
	} else {
		network.Env.Set("SUP_USER", os.Getenv("USER"))
	}

	for _, cmd := range args[1:] {
		l("parse given command: %v", cmd)
		target, isTarget := conf.Targets.Get(cmd)
		l("check if its a target")
		if isTarget {
			for _, cmd := range target {
				command, isCommand := conf.Commands.Get(cmd)
				if !isCommand {
					cmdUsage(conf)
					return nil, nil, fmt.Errorf("%v: %v", entity.ErrCmd, cmd)
				}
				command.Name = cmd
				commands = append(commands, &command)
			}
		}

		// Command?
		l("check if its a command")
		command, isCommand := conf.Commands.Get(cmd)
		if isCommand {
			command.Name = cmd
			commands = append(commands, &command)
		}

		l("check if both searches failed")
		if !isTarget && !isCommand {
			cmdUsage(conf)
			return nil, nil, fmt.Errorf("%v: %v", entity.ErrCmd, cmd)
		}
	}

	return &network, commands, nil
}

func cmdUsage(conf *entity.Supfile) {
	w := &tabwriter.Writer{}
	w.Init(os.Stderr, 4, 4, 2, ' ', 0)
	defer w.Flush()

	// Print available targets/commands.
	fmt.Fprintln(w, "Targets:\t")
	for _, name := range conf.Targets.Names {
		cmds, _ := conf.Targets.Get(name)
		fmt.Fprintf(w, "- %v\t%v\n", name, strings.Join(cmds, " "))
	}

	if conf.Desc != "" {
		fmt.Fprintln(w, "Description:\t")
		fmt.Fprintf(w, "%v", conf.Desc)
	}

	fmt.Fprintln(w, "\t")
	fmt.Fprintln(w, "Commands:\t")
	for _, name := range conf.Commands.Names {
		cmd, _ := conf.Commands.Get(name)
		fmt.Fprintf(w, "- %v\t%v\n", name, cmd.Desc)
	}
	fmt.Fprintln(w)
}

func networkUsage(conf *entity.Supfile) {
	w := &tabwriter.Writer{}
	w.Init(os.Stderr, 4, 4, 2, ' ', 0)
	defer w.Flush()

	// Print available networks/hosts.
	fmt.Fprintln(w, "Networks:\t")
	for _, name := range conf.Networks.Names {
		fmt.Fprintf(w, "- %v\n", name)
		network, _ := conf.Networks.Get(name)
		for _, host := range network.Hosts {
			fmt.Fprintf(w, "\t- %v\n", host.Host)
		}
	}
	fmt.Fprintln(w)
}

func makefileUsage() {
	fmt.Println("No networks defined, makefile mode available")
}
