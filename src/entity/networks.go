package entity

import "gopkg.in/yaml.v2"

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

	n.Names = make([]string, len(items))
	for i, item := range items {
		n.Names[i] = item.Key.(string)
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
