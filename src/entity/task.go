package entity

import (
	"io"
)

// Task represents a set of commands to be run.
type Task struct {
	Run     string
	Input   io.Reader
	Clients []ClientFacade
	TTY     bool
	Sudo    bool `yaml:"sudo" default:"false"`
	Env     EnvList
}
