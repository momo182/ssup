package entity

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
	// Darwin
	return []string{
		"uname", "-s",
	}
}

func (i *Inventory) CheckHomeCommand() []string {
	return []string{
		"pwd",
	}
}

func (i *Inventory) CheckUserCommand() []string {
	return []string{
		"echo", "$USER",
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
