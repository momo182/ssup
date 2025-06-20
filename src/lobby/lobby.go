// Package lobby defines the Lobby object to
// hold a bunch of common objects for reuse
package lobby

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/dump"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/gateway/namespace"
	sf "github.com/wissance/stringFormatter"
	"golang.org/x/crypto/ssh"
)

// ServiceLobby holds common serices used by many places in code
type ServiceLobby struct {
	KeyAuth    *ssh.AuthMethod
	Namespaces *namespace.Namespace
}

// ServiceRegistry holds common serices used by many places in code
var ServiceRegistry *ServiceLobby

var RegisterCmdDisabled = ""

// RegisterCmd is the shell function literal to register a key and value in a file
// later to be parsed by ssup for any envs passed inside
// moved here to reduce code duplication
var RegisterCmdBash = `register() {
echo "will register key '$1' with value '$2'"
local dest="$HOME/.local/ssup/run/_tube_data"
	
if [ "$#" -eq 1 ]; then
    echo "Illegal number of parameters"
	echo "use $0 key value [namespace]"
fi

if [ -n "$SUDO_USER" ]; then
	# shellcheck disable=SC2116
	local PRESUDO_HOME=$(eval echo "~${SUDO_USER}")
	local dest="$PRESUDO_HOME/.local/ssup/run/_tube_data"
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

func insertSSUPCommands(command, homeDir string) string {
	l := kemba.New("lobby::insertSSUPCommands").Printf
	// split input into lines
	lines := strings.Split(command, "\n")

	// create a buffer to store the modified lines
	var buffer bytes.Buffer

	// iterate over the lines and append them to the buffer
	count := 0
	for _, line := range lines {
		l("proccessing line: '%s'", line)
		switch {
		case strings.HasPrefix(line, "#!") && count == 0:
			// we found the shebang, add the source directive after it
			l("found shebang")
			buffer.WriteString(line + "\n")
			newLine := sf.Format("source  \"{0}/.local/ssup/run/_ssup_commands\"", homeDir)
			l("adding source directive: %s", newLine)
			buffer.WriteString(newLine + "\n")
		case !strings.HasPrefix(line, "#!") && count == 0:
			l("found first line w/o shebang")
			newLine := sf.Format("source  \"{0}/.local/ssup/run/_ssup_commands\"", homeDir)
			l("adding source directive: %s", newLine)
			buffer.WriteString(newLine + "\n")
			buffer.WriteString(line + "\n")
		default:
			l("regular line: '%s'", line)
			buffer.WriteString(line + "\n")
		}
		count++
	}

	l("done proccessing, returning:\n%s", buffer.String())
	return buffer.String()
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
func FormatCommandBasedOnSudo(sudo bool, sudoPassword string, Env entity.EnvList, command string, c entity.ClientFacade, task entity.Task, isLocal bool) string {
	l := kemba.New("lobby::FormatCommandBasedOnSudo").Printf
	RegisterCmd := RegisterCmdBash
	var err error

	l("checking for SUP_SUDO")
	RegisterCmd = prepRegisterCommand(command, RegisterCmd)

	inv := c.GetInventory()
	if inv == nil {
		fmt.Println("4D4F7E9F-34F3-483B-9DF3-6B31FE03C39E: no supported shell found")
		os.Exit(1)
	}

	connUserHomeDir := inv.Home
	l("inv.Home: %s", connUserHomeDir)

	hashedPassFile := connUserHomeDir + string(os.PathSeparator) + entity.SSUP_WORK_FOLDER + entity.HASHED_PASS
	mainScriptFile := connUserHomeDir + string(os.PathSeparator) + entity.SSUP_WORK_FOLDER + entity.MAIN_SCRIPT
	envFile := connUserHomeDir + string(os.PathSeparator) + entity.SSUP_WORK_FOLDER + entity.VARS_FILE
	injectedCommands := connUserHomeDir + string(os.PathSeparator) + entity.SSUP_WORK_FOLDER + entity.INJECTED_COMMANDS_FILE
	commandsToRun := insertSSUPCommands(task.Run, connUserHomeDir)
	l("commandsToRun:\n%s", commandsToRun)

	generateBaselineStartSet := func() {
		l("generating _ssup_run: '%s'", commandsToRun)
		if err := c.GenerateOnRemote([]byte(commandsToRun), mainScriptFile); err != nil {
			log.Panic("failed to generate _ssup_run", err)
		}

		l("generating _ssup_env: '%s'", Env.AsExport())
		if err := c.GenerateOnRemote([]byte(Env.AsExport()), envFile); err != nil {
			log.Panic("failed to generate _ssup_env", err)
		}

		l("generating _ssup_commands: '%s'", RegisterCmd)
		if err := c.GenerateOnRemote([]byte(RegisterCmd), injectedCommands); err != nil {
			log.Panic("failed to generate _ssup_commands", err)
		}
	}

	shell := inv.GetShell()
	if shell == "" {
		fmt.Println("6BEB8584-1680-4F81-802E-BAECCC8759CF: no supported shell found")
		os.Exit(1)
	}

	data := map[string]any{
		"arch":              inv.Arch,
		"os_type":           inv.OsType,
		"inv_user":          inv.User,
		"inv_home_folder":   inv.Home,
		"home_folder":       connUserHomeDir,
		"sudo_pass":         sudoPassword,
		"hashed_pass_file":  hashedPassFile,
		"main_script":       mainScriptFile,
		"enrypted_password": sudoPassword,
		"env_file":          envFile,
		"ssup_commands":     injectedCommands,
		"removal_mask":      entity.SSUP_WORK_FOLDER + "_ssup_*",
		"shell":             shell,
	}

	l(dump.Format(data))

	switch sudo {
	case true:
		l("wrapping command into SUDO block:")
		// ENCRYPTION_PASSPHRASE="mystrongpassword" openssl enc -d -aes-256-cbc -pbkdf2 -in ./out.txt   -pass env:ENCRYPTION_PASSPHRASE
		command = sf.FormatComplex(
			"cat {hashed_pass_file} |"+
				" sudo -S {shell} -c \"rm {hashed_pass_file} &&"+
				" echo \"\" && source {env_file} &&"+
				// ^^^^^^ this needs to exist to make sudo prompt go to next line
				" chmod +x {main_script} && {main_script};"+
				" rm -rf {home_folder}/{removal_mask}\"",
			data)
		l("generating remote password file, w pass: %s", sudoPassword)
		generateBaselineStartSet()
		err = c.GenerateOnRemote([]byte(sudoPassword), hashedPassFile)
		if err != nil {
			l("failed to generate remote password file: %s", err)
			return ""
		}

	default:
		l("wrapping command into normal block:")
		command = sf.FormatComplex("{shell} -c 'source {env_file} && chmod +x {main_script} && {main_script}; rm -rf {home_folder}/{removal_mask}'", data)
		generateBaselineStartSet()
	}

	l("done formatting command: %s", command)
	return command
}

func prepRegisterCommand(command string, RegisterCmd string) string {
	l := kemba.New("lobby::prepRegisterCommand").Printf

	lines := strings.Split(command, "\n")
	head := lines[0]
	l("head: %s", head)

	endsOnNu := func(head string) bool {
		return strings.HasSuffix(strings.TrimSpace(head), "/nu")
	}

	searchesNuViaEnv := func(head string) bool {
		return strings.HasSuffix(strings.TrimSpace(head), "/env nu")
	}

	if endsOnNu(head) || searchesNuViaEnv(head) {
		l("dropping register command for nu scripts")
		RegisterCmd = RegisterCmdDisabled
	}

	return RegisterCmd
}

// func encryptPassword(password, encryptionPhrase string) ([]byte, error) {
// 	cmd := exec.Command("openssl", "enc", "-aes-256-cbc", "-salt", "-pbkdf2", "-pass", "env:ENCRYPTION_PASSPHRASE")

// 	stdinPipe, err := cmd.StdinPipe()
// 	if err != nil {
// 		return []byte(""), fmt.Errorf("failed to get stdin pipe: %v", err)
// 	}

// 	go func() {
// 		defer stdinPipe.Close()
// 		stdinPipe.Write([]byte(password))
// 	}()

// 	var stderr bytes.Buffer
// 	cmd.Stderr = &stderr
// 	cmd.Env = []string{"ENCRYPTION_PASSPHRASE=" + encryptionPhrase}

// 	out, err := cmd.Output()
// 	if err != nil {
// 		return []byte(""), fmt.Errorf("command failed: %v, %s", err, stderr.String())
// 	}

// 	return out, nil
// }

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

	nsEnvs := ServiceRegistry.Namespaces.Get(c.GetHost())
	if len(nsEnvs.EnvStore) > 0 {
		for k, v := range nsEnvs.EnvStore {
			l("injecting namespace env: %s=%s", k, v)
			Env.Set(k, v)
		}
	}

	if c.GetTube() != "" {
		l("found tube: %s, attached to host: %s", c.GetTube(), c.GetHost())
		remoteNs := ServiceRegistry.Namespaces.Get(c.GetTube())
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
