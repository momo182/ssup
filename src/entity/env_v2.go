package entity

import (
	"fmt"

	"github.com/clok/kemba"
	"github.com/goforj/godump"
	"gopkg.in/yaml.v2"
)

// EnvVar represents an environment variable
type EnvVar struct {
	Key   string
	Value string
}

func (e EnvVar) String() string {
	return e.Key + `=` + e.Value
}

// AsExport returns the environment variable as a bash export statement
func (e EnvVar) AsExport() string {
	return `export ` + e.Key + `='` + e.Value + `';`
}

// EnvList is a list of environment variables that maps to a YAML map,
// but maintains order, enabling late variables to reference early variables.
type EnvList struct {
	store map[string]string
}

// GetStore returns the whole store
// used in envsubst usecase
func (e EnvList) GetStore() map[string]string {
	return e.store
}

func (e EnvList) Get(key string) string {
	return e.store[key]
}

func (e EnvList) Keys() []string {
	keys := make([]string, 0)
	for key := range e.store {
		keys = append(keys, key)
	}
	return keys
}

func (e EnvList) Slice() []string {
	l := kemba.New("env::slice").Printf
	count := len(e.store)
	l(godump.DumpStr(e.store))
	l("env list length: %v", count)

	envs := make([]string, count)
	count = count - 1
	for key, value := range e.store {
		envVar := EnvVar{
			Key:   key,
			Value: value,
		}
		envs[count] = envVar.String()
		count = +1
	}
	return envs
}

func (e *EnvList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	items := []yaml.MapItem{}

	err := unmarshal(&items)
	if err != nil {
		return err
	}

	e.store = make(map[string]string, 0)
	for _, v := range items {
		e.Set(fmt.Sprintf("%v", v.Key), fmt.Sprintf("%v", v.Value))
	}

	return nil
}

// Set key to be equal value in this list.
func (e *EnvList) Set(key, value string) {
	l := kemba.New("env::set").Printf

	if e.store == nil {
		l("underlying map is nil, creating new map")
		e.store = make(map[string]string, 0)
	}

	l("setting %v = %v", key, value)
	e.store[key] = value
}

func (e *EnvList) AsExport() string {
	// Process all ENVs into a string of form
	// `export FOO="bar"; export BAR="baz";`.
	exports := ``
	for key, value := range *&e.store {
		v := EnvVar{
			Key:   key,
			Value: value,
		}
		exports += v.AsExport() + " "
	}
	return exports
}

// RemoveEnvVarByKey removes an EnvVar from EnvList by key.
// It returns true if the variable was found and removed, or false otherwise.
func (e *EnvList) RemoveEnvVarByKey(key string) bool {
	result := false
	for _, envVar := range e.store {
		if envVar == key {
			delete(e.store, key)
		}
	}
	return result // Key not found
}

func (e *EnvList) String() string {
	if e.store == nil {
		return "{}"
	} else {
		return fmt.Sprintf("%v", e.store)
	}
}
