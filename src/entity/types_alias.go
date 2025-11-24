package entity

import (
	ucutils "github.com/momo182/ssup/src/usecase/utils"
	supfile "github.com/momo182/ssup/src/entity/supfile"
)

// Re-export utility functions from usecase utils to maintain compatibility
func IsShell(cmd string) bool {
	return ucutils.IsShell(cmd)
}

func ResolveShell(value string) (string, error) {
	return ucutils.ResolveShell(value)
}

// Re-export types from subpackages to maintain compatibility
type (
	EnvList = supfile.EnvList
	Supfile = supfile.Supfile
	EnvVar = supfile.EnvVar
	ErrMustUpdate = supfile.ErrMustUpdate
	ErrUnsupportedSupfileVersion = supfile.ErrUnsupportedSupfileVersion
)

// Re-export structs
type (
	NetworkHost = supfile.NetworkHost
	Command     = supfile.Command
	Network     = supfile.Network
	UploadOrder = supfile.UploadOrder
	DownloadOrder = supfile.DownloadOrder
)