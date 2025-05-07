package usecase

import (
	"fmt"
	"os"
	"regexp"

	"github.com/clok/kemba"
	"github.com/mikkeloscar/sshconfig"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/lobby"
	// appinit "github.com/momo182/ssup/src/usecase/appinit"
)

func CheckInitialArgs(network *entity.Network, initialArgs *entity.InitialArgs) {
	l := kemba.New("usecase::CheckInitialArgs").Printf

	l("--only flag filters hosts")
	if initialArgs.OnlyHosts != "" {
		checkOnlyHosts(network, initialArgs)
	}

	l("--except flag filters out hosts")
	if initialArgs.ExceptHosts != "" {
		checkExceptHosts(network, initialArgs)
	}

	l("--sshconfig flag location for ssh_config file")
	if initialArgs.SshConfig != "" {
		checkSSHConfig(network, initialArgs)
	}

}

func checkSSHConfig(network *entity.Network, initialArgs *entity.InitialArgs) {
	l := kemba.New("usecase > checkSSHConfig").Printf

	l("sshconfig: %s", initialArgs.SshConfig)
	confHosts, err := sshconfig.ParseSSHConfig(lobby.ResolvePath(initialArgs.SshConfig))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(41)
	}

	l("forming conf map")
	confMap := map[string]*sshconfig.SSHHost{}
	for _, conf := range confHosts {
		for _, host := range conf.Host {
			confMap[host] = conf
		}
	}

	l("range over hosts and check config map for host present")
	for _, host := range network.Hosts {
		conf, found := confMap[host.Host]
		if found {
			network.User = conf.User
			network.IdentityFile = lobby.ResolvePath(conf.IdentityFile)
			hostPort := []string{fmt.Sprintf("%s:%d", conf.HostName, conf.Port)}
			host := entity.NetworkHost{
				Host: hostPort[0],
			}
			network.Hosts = append(network.Hosts, host)
		}
	}
}

func checkExceptHosts(network *entity.Network, initialArgs *entity.InitialArgs) {
	l := kemba.New("usecase > checkExceptHosts").Printf
	l("prep regexp for --except")
	expr, err := regexp.CompilePOSIX(initialArgs.ExceptHosts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(42)
	}

	l("range over hosts and check for host not present")
	var hosts []entity.NetworkHost
	for _, host := range network.Hosts {
		if !expr.MatchString(host.Host) {
			hosts = append(hosts, host)
		}
	}
	l("found 'only' hosts: %v", len(hosts))
	if len(hosts) == 0 {
		fmt.Fprintln(os.Stderr, fmt.Errorf("no hosts left after --except '%v' regexp", initialArgs.OnlyHosts))
		os.Exit(43)
	}
	network.Hosts = hosts
}

func checkOnlyHosts(network *entity.Network, initialArgs *entity.InitialArgs) {
	l := kemba.New("usecase > checkOnlyHosts").Printf
	l("prep regexp for --only")
	expr, err := regexp.CompilePOSIX(initialArgs.OnlyHosts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(44)
	}

	l("range over hosts and check for host present")
	var hosts []entity.NetworkHost
	for _, host := range network.Hosts {
		if expr.MatchString(host.Host) {
			hosts = append(hosts, host)
		}
	}
	l("found 'only' hosts: %v", len(hosts))
	if len(hosts) == 0 {
		fmt.Fprintln(os.Stderr, fmt.Errorf("no hosts match --only '%v' regexp", initialArgs.OnlyHosts))
		os.Exit(45)
	}
	network.Hosts = hosts
}
