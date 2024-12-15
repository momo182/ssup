package entity

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

type HelpDisplayer struct {
	ShowNetwork bool
	ShowCmd     bool
	// ShowTarget   bool
	ShowMakeMode bool
	Color        bool
}

func (h *HelpDisplayer) Show(conf *Supfile) {
	if h.Color {
		h.printColoredHelp(conf)
		return
	}
	h.printBWHelp(conf)
}

func (h *HelpDisplayer) ShowAll(conf *Supfile) {
	h.ShowNetwork = true
	h.ShowCmd = true
	if h.Color {
		h.printColoredHelp(conf)
		return
	}
	h.printBWHelp(conf)
}

func (h *HelpDisplayer) printColoredHelp(conf *Supfile) {
	introScreen()
	if h.ShowMakeMode {
		colorMakefileUsage()
	}
	if h.ShowNetwork {
		colorNetworkUsage(conf)
	}
	if h.ShowCmd {
		colorCmdUsage(conf)
	}
}

func (h *HelpDisplayer) printBWHelp(conf *Supfile) {
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

func introScreen() {
	// ptermLogo, _ := pterm.DefaultBigText.WithLetters(
	// 	putils.LettersFromStringWithStyle("S", pterm.NewStyle(pterm.FgLightCyan)),
	// 	putils.LettersFromStringWithStyle("SUP", pterm.NewStyle(pterm.FgLightMagenta))).
	// 	Srender()

	// pterm.DefaultCenter.Print(ptermLogo)

	pterm.DefaultHeader.Print(pterm.DefaultHeader.WithTextStyle(pterm.NewStyle(pterm.FgBlack)).WithBackgroundStyle(pterm.NewStyle(pterm.BgWhite)).WithMargin(10).Sprint("SSUP - Super Stackup"))
	fmt.Println("")
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

func colorCmdUsage(conf *Supfile) {
	if conf.Desc != "" {
		pterm.Info.Prefix = pterm.Prefix{Text: "Supfile Description:", Style: pterm.NewStyle(pterm.BgCyan, pterm.FgBlack)}
		pterm.Info.Println(conf.Desc)
		// fmt.Fprintf(w, "%v", conf.Desc)
	}

	fmt.Println("")
	pterm.Info.Prefix = pterm.Prefix{Text: "Commands:", Style: pterm.NewStyle(pterm.BgCyan, pterm.FgBlack)}
	pterm.Info.Println(" ")
	pterm.Println()
	commands := pterm.TableData{{"Command", "Description"}}
	for _, name := range conf.Commands.Names {
		cmd, _ := conf.Commands.Get(name)
		commands = append(commands, []string{name, cmd.Desc})
	}
	pterm.DefaultTable.WithHasHeader(true).WithRowSeparator("-").WithHeaderRowSeparator("-").WithData(commands).Render()

	pterm.Info.Prefix = pterm.Prefix{Text: "Targets:", Style: pterm.NewStyle(pterm.BgCyan, pterm.FgBlack)}
	pterm.Info.Println("")
	pterm.Println()
	targets := pterm.TableData{{"Target", "Commands"}}
	for _, name := range conf.Targets.Names {
		cmds, _ := conf.Targets.Get(name)
		targets = append(targets, []string{name, strings.Join(cmds, " ")})
	}
	pterm.DefaultTable.WithHasHeader(true).WithRowSeparator("-").WithHeaderRowSeparator("-").WithData(targets).Render()
}

func colorNetworkUsage(conf *Supfile) {
	pterm.Info.Prefix = pterm.Prefix{Text: "Networks:", Style: pterm.NewStyle(pterm.BgCyan, pterm.FgBlack)}
	pterm.Info.Println(" ")
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
	root.Text = ""
	if len(networks) > 1 {
		root.Text = "Supfile" // Set the root node text.
	}
	pterm.DefaultTree.WithRoot(root).Render()
	if len(networks) == 0 {
		colorMakefileUsage()
	}
}

func cmdUsage(conf *Supfile) {
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

func networkUsage(conf *Supfile) {
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
