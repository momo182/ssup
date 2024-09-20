package entity

import "gopkg.in/yaml.v2"

// Targets is a list of user-defined targets
type Targets struct {
	Names   []string
	targets map[string][]string
}

func (t *Targets) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(&t.targets)
	if err != nil {
		return err
	}

	var items yaml.MapSlice
	err = unmarshal(&items)
	if err != nil {
		return err
	}

	t.Names = make([]string, len(items))
	for i, item := range items {
		t.Names[i] = item.Key.(string)
	}

	return nil
}

func (t *Targets) Get(name string) ([]string, bool) {
	cmds, ok := t.targets[name]
	return cmds, ok
}

func (t *Targets) Has(name string) bool {
	_, ok := t.targets[name]
	return ok
}
