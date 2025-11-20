package entity

import "fmt"

var InitialArgsStore *InitialArgs = &InitialArgs{}

// InitialArgs contains values from flags during program init
type InitialArgs struct {
	Supfile     string
	EnvVars     FlagStringSlice
	SSHConfig   string
	OnlyHosts   string
	ExceptHosts string

	Debug         bool
	DisablePrefix bool

	ShowVersion     bool
	ShowExample     bool
	ShowHelp        bool
	DisableColor    bool
	TestConnections bool
	CommandArgs     []string
	Force           bool
}

type FlagStringSlice []string

func (f *FlagStringSlice) String() string {
	return fmt.Sprintf("%v", *f)
}

func (f *FlagStringSlice) Set(value string) error {
	*f = append(*f, value)
	return nil
}
