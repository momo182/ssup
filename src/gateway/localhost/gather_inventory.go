package localhost

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/dump"
	"github.com/momo182/ssup/src/entity"
	"golang.org/x/crypto/ssh"
)

func GatherInventory(remote *ssh.Client) (*entity.Inventory, error) {
	l := kemba.New("gw::ssh::gather_inventory").Printf
	inventory := &entity.Inventory{}

	if remote != nil {
		return nil, fmt.Errorf("remote connection is not nil for local client, this should never happen")
	}

	l("Gathering inventory on localhost")

	// check arch command
	archCmd := inventory.DetectArchCommand()
	l("check system arch: %v", archCmd)
	archOutput, err := runLocalCommand(archCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get arch with command: %v", err)
	}
	inventory.Arch = strings.TrimSpace(string(archOutput))

	// check os type command
	osTypeCmd := inventory.CheckOsTypeCommand()
	l("check system arch: %v", osTypeCmd)
	osTypeOutput, err := runLocalCommand(osTypeCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get arch with command: %v", err)
	}
	inventory.OsType = strings.TrimSpace(string(osTypeOutput))

	// Check bash command
	l("check bash installation: %v", inventory.CheckBashCommand())
	bashCmd := inventory.CheckBashCommand()
	bashOutput, err := runLocalCommand(bashCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run bash command: %v", err)
	}
	inventory.Bash = strings.TrimSpace(string(bashOutput)) != ""

	// Check sh command
	l("check sh installation: %v", inventory.CheckShCommand())
	shCmd := inventory.CheckShCommand()
	shOutput, err := runLocalCommand(shCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run sh command: %v", err)
	}
	inventory.Sh = strings.TrimSpace(string(shOutput)) != ""

	// check home command
	homeCmd := inventory.CheckHomeCommand()
	l("check home dir: %v", homeCmd)
	homeOutput, err := runLocalCommand(homeCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get arch with command: %v", err)
	}
	inventory.Home = strings.TrimSpace(string(homeOutput))

	// check user command
	userCmd := inventory.CheckUserCommand()
	l("check user: %v", userCmd)
	userOutput, err := runLocalCommand(userCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get user with command: %v", err)
	}
	inventory.User = strings.TrimSpace(string(userOutput))
	inventory.IsLocal = true

	return inventory, nil
}

func runLocalCommand(command []string) ([]byte, error) {
	l := kemba.New("gw::ssh::run_local_command").Printf

	// both commands to get $HOME and $USER are defined in a not portable way
	// but makefile mode may kinda be useful on windows too
	// so this way, host could be a windows machine
	if command[1] == "-c" {
		command = command[2:]
		l("reduce command to: %v", command)
	}

	if command[0] == "\"echo $HOME\"" {
		home := os.Getenv("HOME")
		l("reploacing home dir: %v", home)
		return []byte(home), nil
	}

	if command[0] == "\"echo $USER\"" {
		user := os.Getenv("USER")
		l("reploacing home dir: %v", user)
		return []byte(user), nil
	}

	// now process the usual case
	cmd := exec.Command(command[0], command[1:]...)
	l(dump.Format(cmd))
	output, err := cmd.Output()
	if err != nil {
		l("got error running command: %v", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Command exited with a non-zero status, but we still want to capture the output
			l("stderr:\n%s", exitErr.Stderr)
			return exitErr.Stderr, nil
		}
		return nil, fmt.Errorf("failed to run command: %v", err)
	}

	l("output: %s", string(output))

	return output, nil
}
