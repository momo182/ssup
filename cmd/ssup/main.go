package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/clok/kemba"
	"github.com/davecgh/go-spew/spew"
	"github.com/goforj/godump"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/gateway/namespace"
	"github.com/momo182/ssup/src/gateway/outrig"
	shellcheckService "github.com/momo182/ssup/src/gateway/shellcheck"
	serviceLobby "github.com/momo182/ssup/src/lobby"
	"github.com/momo182/ssup/src/usecase"
	"github.com/momo182/ssup/src/usecase/appinit"
	_ "github.com/outrigdev/outrig/autoinit"
	oopslogrus "github.com/samber/oops/loggers/logrus"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var initialArgs *entity.InitialArgs = &entity.InitialArgs{}

func init() {
	flag.StringVar(&initialArgs.Supfile, "f", "", "Custom path to ./Supfile[.yml]")
	flag.Var(&initialArgs.EnvVars, "e", "Set environment variables")
	flag.Var(&initialArgs.EnvVars, "env", "Set environment variables")
	flag.StringVar(&initialArgs.SSHConfig, "sshconfig", "", "Read SSH Config file, ie. ~/.ssh/config file")
	flag.StringVar(&initialArgs.OnlyHosts, "only", "", "Filter hosts using regexp")
	flag.StringVar(&initialArgs.ExceptHosts, "except", "", "Filter out hosts using regexp")

	flag.BoolVar(&initialArgs.Debug, "D", false, "Enable debug mode")
	flag.BoolVar(&initialArgs.Debug, "debug", false, "Enable debug mode")
	flag.BoolVar(&initialArgs.DisablePrefix, "disable-prefix", false, "Disable hostname prefix")

	flag.BoolVar(&initialArgs.ShowVersion, "v", false, "Print version")
	flag.BoolVar(&initialArgs.ShowExample, "x", false, "Print eXample Supfile and exit")
	flag.BoolVar(&initialArgs.ShowVersion, "version", false, "Print version")
	flag.BoolVar(&initialArgs.DisableColor, "c", false, "Disable color")
	flag.BoolVar(&initialArgs.DisableColor, "no-color", false, "Disable color")
	flag.BoolVar(&initialArgs.ShowHelp, "h", false, "Show help")
	flag.BoolVar(&initialArgs.ShowHelp, "help", false, "Show help")
	flag.BoolVar(&initialArgs.TestConnections, "t", false, "Test connections")
	flag.BoolVar(&initialArgs.TestConnections, "test", false, "Test connections")
	flag.BoolVar(&initialArgs.Force, "z", false, "Force proceed execution even if some hosts unreachable")

	logrus.SetFormatter(oopslogrus.NewOopsFormatter(&logrus.JSONFormatter{
		PrettyPrint: true,
	}))

	spew.Config.MaxDepth = entity.SPEW_DEPTH
	serviceLobby.ServiceRegistry = &serviceLobby.ServiceLobby{}
	serviceLobby.ServiceRegistry.Namespaces = namespace.New()
	initAuthMethod()
}

func main() {
	l := kemba.New("main").Printf
	flag.Parse()
	initialArgs.CommandArgs = flag.Args()

	outrig.Push("main_command_args", initialArgs)

	if initialArgs.ShowHelp {
		fmt.Fprintln(os.Stderr, entity.ErrUsage, "\n\nOptions:")
		flag.PrintDefaults()
		return
	}

	if initialArgs.ShowExample {
		fmt.Println(entity.ExampleSupfile)
		return
	}

	if initialArgs.ShowVersion {
		fmt.Fprintln(os.Stderr, entity.VERSION)
		return
	}
	l("reading supfile")
	conf := appinit.ReadSupfile(initialArgs)
	if err := unpackInventories(conf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	l("4685D4CC-E39C-49E7-A842-5ECEF446826A: supfile after inv process")
	l("%s", spew.Sdump(conf))
	outrig.Push("main_parsed_supfile", conf)

	l("parse network and commands to be run from args")
	initData := entity.InitState{
		Conf:        conf,
		InitialArgs: initialArgs,
	}

	// MOD here we need to return not just filled single network
	// as it was before
	// in case of target with multiple affixes we
	// have to form playbook

	playbook, err := appinit.ParseInitialArgs(initData)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// negative programming check
	switch {
	case playbook == nil:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	default:
		l("playbook is nil: negative checks passed")
		l("%v", godump.DumpStr(playbook))
	}

	for _, play := range playbook.GetPlays() {
		isMakefileMode := playbook.IsMakefileMode()

		// more nagative programming checks
		switch {
		case play.Network == nil:
			fmt.Fprintln(os.Stderr, fmt.Errorf("got nil pointer when parsing network"))
			os.Exit(3)
		default:
			l("play is not nil: negative checks passed")
		}

		network := play.Network
		commands := play.Commands
		l("starting play run\nhosts: %v\ncommands: %v\n", len(network.Hosts), len(commands))

		if e := shellcheckService.RunShellcheck(conf); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(42)
		}

		usecase.CheckInitialArgs(network, initialArgs)
		vars := appinit.MergeVars(conf, network)

		l("parse CLI --env flag env vars") // define $SUP_ENV and override values defined in Supfile.
		appinit.SetEnvValues(&vars, initialArgs)
		appinit.GenerateSUPENVFrom(&vars)

		l("create new Stackup app")
		app := usecase.NewStackup(conf)

		app.Debug(initialArgs.Debug)
		app.Prefix(!initialArgs.DisablePrefix)
		app.Args = initialArgs

		l("run all the commands in the given network")
		runOrder := entity.RunOrder{
			MakefileMode: isMakefileMode,
			Network:      network,
			Vars:         vars,
			Commands:     commands,
			Args:         *initialArgs,
		}

		err = app.Run(runOrder)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
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

// initAuthMethod initiates SSH authentication method.
func initAuthMethod() {
	l := kemba.New("lobby::initAuthMethod").Printf
	l("initializing SSH authentication method")
	var signers []ssh.Signer

	// If there's a running SSH Agent, try to use its Private keys.
	l("check ssh agent is running")
	sock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err == nil {
		l("using SSH Agent")
		agent := agent.NewClient(sock)
		signers, _ = agent.Signers()
	}

	// Try to read user's SSH private keys from the standard paths.
	l("check if user has SSH private keys")
	files, _ := filepath.Glob(os.Getenv("HOME") + "/.ssh/id_*")
	for _, file := range files {
		if strings.HasSuffix(file, ".pub") {
			continue // Skip public keys.
		}
		data, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			continue
		}
		l("Using SSH private key: %s", file)
		signers = append(signers, signer)

	}
	l("found %v SSH signers", len(signers))
	auth := ssh.PublicKeys(signers...)
	serviceLobby.ServiceRegistry.KeyAuth = auth
	l("done initializing SSH authentication method")
}
