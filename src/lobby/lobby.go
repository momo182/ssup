package lobby

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/dump"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/gateway/namespace"
	"github.com/momo182/ssup/src/gateway/shellcheck"
	sf "github.com/wissance/stringFormatter"
	"golang.org/x/crypto/ssh"
)

// ServiceLobby
type ServiceLobby struct {
	KeyAuth    *ssh.AuthMethod
	Shellcheck *shellcheck.ShellCheck
	Namespaces *namespace.Namespace
}

// Lobby holds common serices used by many places in code
var Lobby *ServiceLobby

// RegisterCmd is the shell function literal to register a key and value in a file
// later to be parsed by ssup for any envs passed inside
// moved here to reduce code duplication
var RegisterCmd = `register() {
local dest="$HOME/_ssup_vars.env"
	
if [ "$#" -eq 1 ]; then
    echo "Illegal number of parameters"
	echo "use $0 key value [namespace]"
fi

if [ -n "$SUDO_USER" ]; then
	# shellcheck disable=SC2116
	local PRESUDO_HOME=$(eval echo "~${SUDO_USER}")
	local dest="$PRESUDO_HOME/_ssup_vars.env"
fi

if [ "$#" -eq 2 ]; then
	local key=$1
	local val=$2
	echo "${key}=${val}" >> "$dest"
fi
	
if [ "$#" -eq 3 ]; then
	local key=$1
	local val=$2
	local namespace="$3"
	echo "${namespace} ${key}=${val}" >> "$dest"
fi
}
`

// MustFindRclone returns the path to rclone
// if it is not found, it will exit with code 11
func MustFindRclone() string {
	rclone, e := exec.LookPath("rclone")
	if e != nil {
		fmt.Println("Please install rclone on your system, and make it available in $PATH")
		os.Exit(11)
	}
	return rclone
}

// FormatCommandBasedOnSudo returns the command to be executed
//
// `register` bash function is injected here, injected only for non sudo invocations
// if sudo is set to true, the command will be wrapped into script
// and and remote command will just execute that script
//
// which means we have to inject the command into a script for sudo invocation too
// and that happens to hapen inside
// func (c *SSHClient) GenerateOnRemote(data []byte) error
// which shares the same code defined in lobby.RegisterCmd
func FormatCommandBasedOnSudo(sudo bool, sudoPassword string, Env entity.EnvList, exportCmd string, scriptName string, command string, c entity.ClientFacade, task entity.Task) string {
	l := kemba.New("lobby::FormatCommandBasedOnSudo").Printf
	l("checking for SUP_SUDO")
	switch sudo {
	case true:
		l("wrapping command into SUDO block:")
		data := map[string]any{
			"sudo_pass":        sudoPassword,
			"env_setup":        Env.AsExport(),
			"export_command":   exportCmd,
			"ssup_script_name": scriptName,
			"vars_tail":        entity.VARS_TAIL,
		}
		command = sf.FormatComplex("echo {sudo_pass} | sudo -S bash -c '{env_setup} chmod +x ./{ssup_script_name};bash ./{ssup_script_name}; rm -rf ./*{ssup_script_name}'", data)

		if err := c.GenerateOnRemote([]byte(task.Run)); err != nil {
			log.Panic("failed to generate remote command", err)
		}

	default:
		data := map[string]any{
			"command":        strings.TrimSpace(command) + "\n\n",
			"env_setup":      Env.AsExport(),
			"export_command": exportCmd,
			"vars_tail":      entity.VARS_TAIL,
		}
		command = sf.FormatComplex("{env_setup}{command}", data)
	}
	l("done formatting command: %s", command)
	return command
}

// InjectNamespacesAndEnvs injects the namespaces and envs into the command
//
// namespaces follow command definitions to overwrite envs with values returned
// from namespaces
func InjectNamespacesAndEnvs(task entity.Task, Env entity.EnvList, c entity.ClientFacade) entity.EnvList {
	l := kemba.New("lobby::InjectNamespacesAndEnvs").Printf

	if len(task.Env.Keys()) > 0 {
		for _, key := range task.Env.Keys() {
			l("injecting task env: %s=%s", key, task.Env.Get(key))

			Env.Set(key, task.Env.Get(key))
		}
	}

	nsEnvs := Lobby.Namespaces.Get(c.GetHost())
	if len(nsEnvs.EnvStore) > 0 {
		for k, v := range nsEnvs.EnvStore {
			l("injecting namespace env: %s=%s", k, v)
			Env.Set(k, v)
		}
	}

	if c.GetTube() != "" {
		l("found tube: %s, attached to host: %s", c.GetTube(), c.GetHost())
		remoteNs := Lobby.Namespaces.Get(c.GetTube())
		if len(remoteNs.EnvStore) > 0 {
			for k, v := range remoteNs.EnvStore {
				l("injecting namespace env: %s=%s", k, v)
				Env.Set(k, v)
			}
		}
	}

	l("done injecting namespaces and envs:\n%s", dump.Format(Env))
	return Env
}
