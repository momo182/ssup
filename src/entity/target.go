package entity

import (
	"strings"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/dump"
	"gopkg.in/yaml.v2"
)

type AffixMappig struct {
	TargetName     string
	AffixedNetwork string
	CommandName    string
}

// Targets is a list of user-defined targets
// by default affixed are mapped by command name
type Targets struct {
	Names   []string
	targets map[string][]string
	affixes map[string]AffixMappig
}

func (t *Targets) UnmarshalYAML(unmarshal func(interface{}) error) error {
	l := kemba.New("entity::targets::UnmarshalYAML").Printf
	err := unmarshal(&t.targets)

	if err != nil {
		return err
	}

	var items yaml.MapSlice
	err = unmarshal(&items)

	if err != nil {
		return err
	}

	l("dump: 4DE17602-DD0A-4BB7-9F5C-5791CD88C2DB")
	l("%s", dump.Format(items))

	t.Names = make([]string, len(items))

	if t.affixes == nil {
		t.affixes = make(map[string]AffixMappig)
	}

	for i, item := range items {
		value := item.Value.([]interface{})
		l("value: %s", value)

		// if we split and it has two parts
		// then first is the command and second is the network
		for _, part := range value {
			l("part: %s", part)
			parts := strings.Split(part.(string), " ")
			if len(parts) == 2 {
				// command := parts[0]
				// network := parts[1]
				mapping := AffixMappig{
					TargetName:     item.Key.(string),
					AffixedNetwork: parts[1],
					CommandName:    parts[0],
				}
				t.Names[i] = item.Key.(string)
				t.affixes[parts[0]] = mapping

				for counterOfTarget, targetToSearch := range t.targets {
					for counterOfCommand, commandToRun := range targetToSearch {
						if strings.Contains(commandToRun, mapping.CommandName) {
							t.targets[counterOfTarget][counterOfCommand] = mapping.CommandName
						}
					}

				}

				continue
			}
		}

		t.Names[i] = item.Key.(string)

	}

	l("dump: 2C5FF6CA-FA5A-465B-A2C7-D40E94652819")
	l("%s", dump.Format(t))
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

func (t *Targets) GetAffixByCommandName(name string) (AffixMappig, bool) {
	affix, ok := t.affixes[name]
	return affix, ok
}

func (t *Targets) HasAffixes() bool {
	return len(t.affixes) > 0
}
