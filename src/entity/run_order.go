package entity

import "github.com/momo182/ssup/src/entity/supfile"

type RunOrder struct {
	MakefileMode bool
	Network      *supfile.Network
	Vars         supfile.EnvList
	Commands     []*supfile.Command
	Args         InitialArgs
}
