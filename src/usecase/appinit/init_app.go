package appinit

import (
	"flag"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
	keys "github.com/momo182/ssup/src/gateway/keyauthprovider"
	oopslogrus "github.com/samber/oops/loggers/logrus"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func InitApp() {
	l := kemba.New("uc::appinit::InitApp").Printf
	l("setting cli flags")
	flag.StringVar(&entity.InitialArgsStore.Supfile, "f", "", "Custom path to ./Supfile[.yml]")
	flag.Var(&entity.InitialArgsStore.EnvVars, "e", "Set environment variables")
	flag.Var(&entity.InitialArgsStore.EnvVars, "env", "Set environment variables")
	flag.StringVar(&entity.InitialArgsStore.SSHConfig, "sshconfig", "", "Read SSH Config file, ie. ~/.ssh/config file")
	flag.StringVar(&entity.InitialArgsStore.OnlyHosts, "only", "", "Filter hosts using regexp")
	flag.StringVar(&entity.InitialArgsStore.ExceptHosts, "except", "", "Filter out hosts using regexp")

	flag.BoolVar(&entity.InitialArgsStore.Debug, "D", false, "Enable debug mode")
	flag.BoolVar(&entity.InitialArgsStore.Debug, "debug", false, "Enable debug mode")
	flag.BoolVar(&entity.InitialArgsStore.DisablePrefix, "disable-prefix", false, "Disable hostname prefix")

	flag.BoolVar(&entity.InitialArgsStore.ShowVersion, "v", false, "Print version")
	flag.BoolVar(&entity.InitialArgsStore.ShowExample, "x", false, "Print eXample Supfile and exit")
	flag.BoolVar(&entity.InitialArgsStore.ShowVersion, "version", false, "Print version")
	flag.BoolVar(&entity.InitialArgsStore.DisableColor, "c", false, "Disable color")
	flag.BoolVar(&entity.InitialArgsStore.DisableColor, "no-color", false, "Disable color")
	flag.BoolVar(&entity.InitialArgsStore.ShowHelp, "h", false, "Show help")
	flag.BoolVar(&entity.InitialArgsStore.ShowHelp, "help", false, "Show help")
	flag.BoolVar(&entity.InitialArgsStore.TestConnections, "t", false, "Test connections")
	flag.BoolVar(&entity.InitialArgsStore.TestConnections, "test", false, "Test connections")
	flag.BoolVar(&entity.InitialArgsStore.Force, "z", false, "Force proceed execution even if some hosts unreachable")

	logrus.SetFormatter(oopslogrus.NewOopsFormatter(&logrus.JSONFormatter{
		PrettyPrint: true,
	}))

	initAuthMethod()
	l("all flags set")
}

// initAuthMethod initiates SSH authentication method.
func initAuthMethod() {
	l := kemba.New("uc::initAuthMethod").Printf
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
	keys.KeyAuth = auth
	l("done initializing SSH authentication method")
}
