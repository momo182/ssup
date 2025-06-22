package entity

import (
	"fmt"
	"os"
)

type Inventory struct {
	Bash    bool
	Sh      bool
	Arch    string
	Home    string
	User    string
	OsType  string
	IsLocal bool
}

func (i *Inventory) CheckBashCommand() []string {
	return []string{
		"which", "bash",
	}
}

func (i *Inventory) CheckShCommand() []string {
	return []string{
		"which", "sh",
	}
}

func (i *Inventory) CheckOsTypeCommand() []string {
	// Darwin or Linux
	return []string{
		"uname", "-s",
	}
}

// CheckHomeCommand returns the command to get the home directory
// based on the OS type
//
// it's expected that it will run AFTER a call to check os type
func (i *Inventory) CheckHomeCommand() []string {
	// negative checks
	// if not set, fail now as we cant get the home dir
	// if we dont know an OS type
	if i.OsType == "" {
		fmt.Println("unable to determine os type, cannot get home")
		os.Exit(1)
	}

	// if not Darwin or Linux assume its windows
	if i.OsType != "Darwin" && i.OsType != "Linux" {
		return i.GetHomeWinCommand()
	}

	return i.GetHomeUnixCommand()
}

func (i *Inventory) CheckUserCommand() []string {
	return []string{
		i.GetShell(), "-c", "\"echo $USER\"",
	}
}

func (i *Inventory) DetectArchCommand() []string {

	return []string{
		"uname", "-m",
	}
}

func (i *Inventory) GetShell() string {
	if i.Bash {
		return "bash"
	}

	if i.Sh {
		return "sh"
	}

	return ""
}

func (i *Inventory) GetHomeUnixCommand() []string {
	return []string{
		i.GetShell(), "-c", "\"echo $HOME\"",
	}
}

func (i *Inventory) GetHomeWinCommand() []string {
	return []string{
		"echo", "%USERPROFILE%",
	}
}
