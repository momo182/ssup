package usecase

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
)

func ensureNetworkExists(netName string, conf *entity.Supfile, helpMenu entity.HelpDisplayer) {
	_, ok := conf.Networks.Get(netName)
	if !ok {
		helpMenu.ShowNetwork = true
		helpMenu.Show(conf)
		fmt.Printf("Network: '%s' does not exist", netName)
		os.Exit(25)
	}
}

func ensureNetworkNonempty(network entity.Network, helpMenu entity.HelpDisplayer, conf *entity.Supfile) {
	if len(network.Hosts) == 0 {
		helpMenu.ShowNetwork = true
		helpMenu.Show(conf)
	}
}

func overrideEnvFromArgs(envFromArgs entity.FlagStringSlice, network *entity.Network) {
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

func addSSUPDefaultEnvs(network *entity.Network, args []string) {
	l := kemba.New("usecase::addSSUPDefaultEnvs").Printf

	switch {
	case network == nil:
		fmt.Printf("ERR: 597867DE-D399-4EF0-8B51-251760D058A7 network is nil")
		os.Exit(3)
	default:
		l("network negative checks passed")
	}

	l("add default env variable with current network")
	switch {
	case network.Name == "localhost":
		network.Env.Set("SUP_NETWORK", "localhost")
	default:
		network.Env.Set("SUP_NETWORK", args[0])
	}

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
