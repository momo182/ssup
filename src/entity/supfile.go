package entity

import (
	"fmt"
)

// Supfile represents the Stack Up configuration YAML file.
type Supfile struct {
	Networks Networks `yaml:"networks"`
	Commands Commands `yaml:"commands"`
	Targets  Targets  `yaml:"targets"`
	Env      EnvList  `yaml:"env"`
	Version  string   `yaml:"version"`
	Desc     string   `yaml:"desc"`
}

func (s Supfile) GetNetworkByName(name string) (*Network, error) {
	for _, network := range s.Networks.Names {
		if network == name {
			result, ok := s.Networks.Get(network)
			if !ok {
				return nil, fmt.Errorf("network %q not found", name)
			}
			return &result, nil
		}
	}
	return nil, fmt.Errorf("network %q not found", name)

}

type ErrMustUpdate struct {
	Msg string
}

type ErrUnsupportedSupfileVersion struct {
	Msg string
}

func (e ErrMustUpdate) Error() string {
	return fmt.Sprintf("%v\n\nPlease update sup by `go get -u github.com/pressly/sup/cmd/sup`", e.Msg)
}

func (e ErrUnsupportedSupfileVersion) Error() string {
	return fmt.Sprintf("%v\n\nCheck your Supfile version (available latest version: v0.5)", e.Msg)
}
