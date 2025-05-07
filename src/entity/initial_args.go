package entity

import "fmt"

type InitialArgs struct {
	Supfile     string
	EnvVars     FlagStringSlice
	SshConfig   string
	OnlyHosts   string
	ExceptHosts string

	Debug         bool
	DisablePrefix bool

	ShowVersion  bool
	ShowHelp     bool
	DisableColor bool
	CommandArgs  []string
}

type FlagStringSlice []string

func (f *FlagStringSlice) String() string {
	return fmt.Sprintf("%v", *f)
}

func (f *FlagStringSlice) Set(value string) error {
	*f = append(*f, value)
	return nil
}
