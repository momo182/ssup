package entity

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Networks is a list of user-defined networks
type Networks struct {
	Names []string
	Nets  map[string]Network
}

func (n *Networks) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(&n.Nets)
	if err != nil {
		return err
	}

	var items yaml.MapSlice
	err = unmarshal(&items)
	if err != nil {
		return err
	}

	// TODO fix this
	n.Names = make([]string, len(items))
	for i, item := range items {
		// dump.Print(item)

		netName := item.Key.(string)
		// network := item.Value.(yaml.MapItem)
		n.Names[i] = netName
		thisNet, ok := n.Get(netName)
		if !ok {
			fmt.Printf("ERR: 03A4FF08-3E6A-4AD5-8A49-B0A1A8AFCD96"+
				"failed to get network: %s\n", netName)
			os.Exit(1)
		}

		thisNet.Name = netName
		// dump.Print(thisNet)
		n.Nets[netName] = thisNet

	}

	return nil
}

func (n *Networks) Get(name string) (Network, bool) {
	net, ok := n.Nets[name]
	return net, ok
}

// this is just to set localhost
// so we dont process slice of values
// as we should be...
func (n *Networks) Set(name string, value string) {
	h := NetworkHost{
		Host: value,
	}
	n.Names = append(n.Names, name)
	n.Nets = map[string]Network{}
	n.Nets[name] = Network{
		Hosts: []NetworkHost{h},
	}
}
