package entity

import "github.com/momo182/ssup/src/entity/supfile"

type InitState struct {
	Conf        *supfile.Supfile
	InitialArgs *InitialArgs
}
