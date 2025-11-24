package entity

import (
	"io"

	"github.com/momo182/ssup/src/entity/supfile"
)

// Task represents a set of commands to be run.
type Task struct {
	Run     string
	Input   io.Reader
	Clients []ClientFacade
	TTY     bool
	Sudo    bool `yaml:"sudo" default:"false"`
	Env     supfile.EnvList
	// num     int
	Id   string
	Dist string
}
