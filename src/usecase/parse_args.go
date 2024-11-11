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
	"github.com/no-src/nsgo/osutil"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

type helpDisplayer struct {
	ShowNetwork bool
	ShowCmd     bool
	// ShowTarget   bool
	ShowMakeMode bool
	Color        bool
}

func (h *helpDisplayer) Show(conf *entity.Supfile) {
	if h.Color {
		h.printColoredHelp(conf)
		return
	}
	h.printBWHelp(conf)
}

func (h *helpDisplayer) printColoredHelp(conf *entity.Supfile) {
	introScreen()
	if h.ShowMakeMode {
		colorMakefileUsage()
	}
	if h.ShowNetwork {
		ColorNetworkUsage(conf)
	}
	if h.ShowCmd {
		ColorCmdUsage(conf)
	}
}

func (h *helpDisplayer) printBWHelp(conf *entity.Supfile) {
	if h.ShowMakeMode {
		makefileUsage()
	}
	if h.ShowNetwork {
		networkUsage(conf)
	}
	if h.ShowCmd {
		cmdUsage(conf)
	}
}

// ParseInitialArgs parses args and returns network and commands to be run.
// On error, it prints usage and exits.
func ParseInitialArgs(conf *entity.Supfile, envFromArgs entity.FlagStringSlice) (*entity.Network, []*entity.Command, error) {
	var commands []*entity.Command
	args := flag.Args()
	helpMenu := helpDisplayer{}
	helpMenu.Color = true

	// dont trust windows on colors, nushell over ssh played bad
	// don't expect this will be better
	if osutil.IsWindows() {
		helpMenu.Color = false
	}

	l := kemba.New("usecase::ParseInitialArgs").Printf
	l("check args len")

	switch {
	case len(args) < 1: // no args given
		// networkUsage(conf)
		// cmdUsage(conf)
		helpMenu.ShowNetwork = true
		helpMenu.ShowCmd = true
		helpMenu.Show(conf)
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

					// networkUsage(conf)
					// cmdUsage(conf)
					helpMenu.ShowNetwork = true
					helpMenu.ShowCmd = true
					helpMenu.Show(conf)
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
		// networkUsage(conf)
		helpMenu.ShowNetwork = true
		helpMenu.Show(conf)
		return nil, nil, entity.ErrUnknownNetwork
	}

	l("parse CLI --env flag env vars, override values defined in Network env")
	overrideEnvFromArgs(envFromArgs, network)

	l("parse inventory")
	hosts, err := network.ParseInventory()
	if err != nil {
		return nil, nil, err
	}
	network.Hosts = append(network.Hosts, hosts...)

	l("does the <network> have at least one host?")
	if len(network.Hosts) == 0 {
		// networkUsage(conf)
		helpMenu.ShowNetwork = true
		helpMenu.Show(conf)
		return nil, nil, entity.ErrNetworkNoHosts
	}

	l("check for the second argument")
	if len(args) < 2 {
		// cmdUsage(conf)
		helpMenu.ShowCmd = true
		helpMenu.Show(conf)
		return nil, nil, entity.ErrUsage
	}

	addSSUPDefaultEnvs(network, args)

	for _, cmd := range args[1:] {
		target, isTarget := conf.Targets.Get(cmd)
		l("parse given command: %v", cmd)
		l("check if its a target")

		if isTarget {
			for _, cmd := range target {
				command, isCommand := conf.Commands.Get(cmd)
				if !isCommand {
					// cmdUsage(conf)
					helpMenu.ShowCmd = true
					helpMenu.Show(conf)
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
			// cmdUsage(conf)
			helpMenu.ShowCmd = true
			helpMenu.Show(conf)
			return nil, nil, fmt.Errorf("%v: %v", entity.ErrCmd, cmd)
		}
	}

	return &network, commands, nil
}

func addSSUPDefaultEnvs(network entity.Network, args []string) {
	l := kemba.New("usecase::ParseInitialArgs::addSSUPDefaultEnvs").Printf
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
}

func overrideEnvFromArgs(envFromArgs entity.FlagStringSlice, network entity.Network) {
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

func colorMakefileUsage() {
	// pterm.Info.Prefix = pterm.Prefix{Text: "Make mode", Style: pterm.NewStyle(pterm.BgCyan, pterm.FgBlack)}
	pterm.Warning.Println("No networks defined" +
		"\n@makefile mode available !!!!")
	pterm.Println()
}

func ColorCmdUsage(conf *entity.Supfile) {

	if conf.Desc != "" {
		pterm.Info.Prefix = pterm.Prefix{Text: "Supfile Description:", Style: pterm.NewStyle(pterm.BgCyan, pterm.FgBlack)}
		pterm.Info.Println(conf.Desc)
		// fmt.Fprintf(w, "%v", conf.Desc)
	}

	fmt.Println("")
	pterm.Info.Prefix = pterm.Prefix{Text: "Commands:", Style: pterm.NewStyle(pterm.BgCyan, pterm.FgBlack)}
	pterm.Info.Println("all the commands found:")
	pterm.Println()
	commands := pterm.TableData{{"Command", "Description"}}
	for _, name := range conf.Commands.Names {
		cmd, _ := conf.Commands.Get(name)
		commands = append(commands, []string{name, cmd.Desc})
	}
	pterm.DefaultTable.WithHasHeader(true).WithRowSeparator("-").WithHeaderRowSeparator("-").WithData(commands).Render()

	pterm.Info.Prefix = pterm.Prefix{Text: "Targets:", Style: pterm.NewStyle(pterm.BgCyan, pterm.FgBlack)}
	pterm.Info.Println("all the targets found:")
	pterm.Println()
	targets := pterm.TableData{{"Target", "Commands"}}
	for _, name := range conf.Targets.Names {
		cmds, _ := conf.Targets.Get(name)
		targets = append(targets, []string{name, strings.Join(cmds, " ")})
	}
	pterm.DefaultTable.WithHasHeader(true).WithRowSeparator("-").WithHeaderRowSeparator("-").WithData(targets).Render()
}

func ColorNetworkUsage(conf *entity.Supfile) {
	pterm.Info.Prefix = pterm.Prefix{Text: "Networks:", Style: pterm.NewStyle(pterm.BgCyan, pterm.FgBlack)}
	pterm.Info.Println("all the networks found:")
	pterm.Println()
	networks := pterm.LeveledList{}
	for _, name := range conf.Networks.Names {
		// fmt.Fprintf(w, "- %v\n", name)
		networks = append(networks, pterm.LeveledListItem{
			Level: 0,
			Text:  name,
		})
		network, _ := conf.Networks.Get(name)
		for _, host := range network.Hosts {
			// fmt.Fprintf(w, "\t- %v\n", host.Host)
			networks = append(networks, pterm.LeveledListItem{
				Level: 1,
				Text:  host.Host,
			})
		}
	}
	root := putils.TreeFromLeveledList(networks)
	root.Text = "none"
	if len(networks) > 1 {
		root.Text = "Supfile" // Set the root node text.
	}
	pterm.DefaultTree.WithRoot(root).Render()
	if len(networks) == 0 {
		colorMakefileUsage()
	}
}

func introScreen() {
	ptermLogo, _ := pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithStyle("S", pterm.NewStyle(pterm.FgLightCyan)),
		putils.LettersFromStringWithStyle("SUP", pterm.NewStyle(pterm.FgLightMagenta))).
		Srender()

	pterm.DefaultCenter.Print(ptermLogo)

	pterm.DefaultCenter.Print(pterm.DefaultHeader.WithFullWidth().WithTextStyle(pterm.NewStyle(pterm.FgBlack)).WithBackgroundStyle(pterm.NewStyle(pterm.BgWhite)).WithMargin(10).Sprint("SSUP - Super Stackup"))
}
