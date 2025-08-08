package ssh

import (
	"fmt"
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

	l("Gathering inventory on %v", remote.RemoteAddr())

	invCmd := []string{}
	invCmd = append(invCmd, inventory.DetectArchCommand()...)
	invCmd = append(invCmd, ";")
	invCmd = append(invCmd, []string{"echo", "-n", "^"}...)
	invCmd = append(invCmd, ";")
	invCmd = append(invCmd, inventory.CheckOsTypeCommand()...)
	invCmd = append(invCmd, ";")
	invCmd = append(invCmd, []string{"echo", "-n", "^"}...)
	invCmd = append(invCmd, ";")
	invCmd = append(invCmd, inventory.CheckBashCommand()...)
	invCmd = append(invCmd, ";")
	invCmd = append(invCmd, []string{"echo", "-n", "^"}...)
	invCmd = append(invCmd, ";")
	invCmd = append(invCmd, inventory.CheckShCommand()...)

	l("check system arch: %v", invCmd)
	archOutput, err := runRemoteCommand(remote, invCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run sh command: %v", err)
	}
	parts := strings.Split(string(archOutput), "^")
	l(dump.Format(parts))

	inventory.Bash = false
	inventory.Sh = false
	inventory.Arch = parts[0]
	inventory.OsType = parts[1]
	if parts[2] != "" {
		inventory.Bash = true
	}
	if parts[3] != "" {
		inventory.Sh = true
	}

	// // check os type command
	// osTypeCmd := inventory.CheckOsTypeCommand()
	// l("check os type: %v", osTypeCmd)
	// osTypeOutput, err := runRemoteCommand(remote, osTypeCmd)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to run sh command: %v", err)
	// }
	// inventory.OsType = strings.TrimSpace(string(osTypeOutput))

	// // Check bash command
	// l("check bash installation: %v", inventory.CheckBashCommand())
	// bashCmd := inventory.CheckBashCommand()
	// bashOutput, err := runRemoteCommand(remote, bashCmd)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to run bash command: %v", err)
	// }
	// inventory.Bash = strings.TrimSpace(string(bashOutput)) != ""

	// // Check sh command
	// l("check sh installation: %v", inventory.CheckShCommand())
	// shCmd := inventory.CheckShCommand()
	// shOutput, err := runRemoteCommand(remote, shCmd)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to run sh command: %v", err)
	// }
	// inventory.Sh = strings.TrimSpace(string(shOutput)) != ""

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

func runRemoteCommand(remote *ssh.Client, command []string) (string, error) {
	l := kemba.New("gw::ssh::run_remote_command").Printf

	session, err := remote.NewSession()
	defer session.Close()

	fullCommand := strings.Join(command, " ")
	l("run command: '%v'", fullCommand)
	output, err := session.Output(fullCommand)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Command exited with a non-zero status, but we still want to capture the output
			return string(exitErr.Stderr), nil
		}
		return "", fmt.Errorf("failed to run command: %v", err)
	}
	rawData := string(output)
	clean1 := strings.Replace(rawData, "\n", "", -1)
	clean2 := strings.Replace(clean1, "\r", "", -1)
	// inventory.Arch = strings.TrimSpace(clean2)
	l("output: %s", string(clean2))

	return clean2, nil
}
