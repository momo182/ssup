package ssh

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
	"golang.org/x/crypto/ssh"
)

func GatherInventory(remote *ssh.Client) (*entity.Inventory, error) {
	l := kemba.New("gw::ssh::gather_inventory").Printf
	inventory := &entity.Inventory{}

	l("Gathering inventory on %v", remote.RemoteAddr())
	// Check bash command
	l("check bash installation: %v", inventory.CheckBashCommand())
	bashCmd := inventory.CheckBashCommand()
	bashOutput, err := runRemoteCommand(remote, bashCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run bash command: %v", err)
	}
	inventory.Bash = strings.TrimSpace(string(bashOutput)) != ""

	// Check sh command
	l("check sh installation: %v", inventory.CheckShCommand())
	shCmd := inventory.CheckShCommand()
	shOutput, err := runRemoteCommand(remote, shCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run sh command: %v", err)
	}
	inventory.Sh = strings.TrimSpace(string(shOutput)) != ""

	// check arch command
	archCmd := inventory.DetectArchCommand()
	l("check system arch: %v", archCmd)
	archOutput, err := runRemoteCommand(remote, archCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run sh command: %v", err)
	}
	inventory.Arch = strings.TrimSpace(string(archOutput))

	// check os type command
	osTypeCmd := inventory.CheckOsTypeCommand()
	l("check os type: %v", osTypeCmd)
	osTypeOutput, err := runRemoteCommand(remote, osTypeCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run sh command: %v", err)
	}
	inventory.OsType = strings.TrimSpace(string(osTypeOutput))

	// check home command
	homeCmd := inventory.CheckHomeCommand()
	l("check home dir: %v", homeCmd)
	homeOutput, err := runRemoteCommand(remote, homeCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run sh command: %v", err)
	}
	inventory.Home = strings.TrimSpace(string(homeOutput))

	// check user command
	userCmd := inventory.CheckUserCommand()
	l("check user: %v", userCmd)
	userOutput, err := runRemoteCommand(remote, userCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run sh command: %v", err)
	}
	inventory.User = strings.TrimSpace(string(userOutput))

	return inventory, nil
}

func runRemoteCommand(remote *ssh.Client, command []string) ([]byte, error) {
	l := kemba.New("gw::ssh::run_remote_command").Printf
	session, err := remote.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()
	l("run command: %v", command)
	output, err := session.Output(strings.Join(command, " "))
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Command exited with a non-zero status, but we still want to capture the output
			return exitErr.Stderr, nil
		}
		return nil, fmt.Errorf("failed to run command: %v", err)
	}
	l("output: %s", string(output))

	return output, nil
}
