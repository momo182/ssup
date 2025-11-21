package entity

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/clok/kemba"
	"github.com/goforj/godump"
	"github.com/gookit/goutil/strutil"
	"github.com/momo182/ssup/src/gateway/outrig"
)

// Network is group of hosts with extra custom env vars.
type Network struct {
	Env       EnvList       `yaml:"env"`
	Inventory string        `yaml:"inventory"`
	Hosts     []NetworkHost `yaml:"hosts"`
	Bastion   string        `yaml:"bastion"` // Jump host for the environment

	// Should these live on Hosts too? We'd have to change []string to struct, even in Supfile.
	User         string `yaml:"user"`
	Password     string `yaml:"pass" `
	IdentityFile string `yaml:"id_file"`
	Name         string
}

func (n Network) String() string {
	if n.Name != "" {
		lines := []string{}
		lines = append(lines, "network: "+n.Name)
		lines = append(lines, "  envs:")
		for k, v := range n.Env.store {
			if v != "" {
				lines = append(lines, "    "+k+": "+v)
			}
		}
		lines = append(lines, "  hosts:")
		for _, h := range n.Hosts {
			if h.Host != "" {
				lines = append(lines, "    +-"+h.Host)
			}
		}
		return strings.Join(lines, "\n")
	} else {
		return "network:"
	}
}

// ParseInventory runs the inventory command, if provided, and appends
// the command's output lines to the manually defined list of hosts.
func (n Network) ParseInventory() ([]NetworkHost, error) {
	l := kemba.New("Network.ParseInventory").Printf
	l("inventory: %s", n.Inventory)
	if n.Inventory == "" {
		l("no inventory given")
		return nil, nil
	}

	l("forming command")
	cmd := exec.Command("/bin/sh", "-c", n.Inventory)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, n.Env.Slice()...)
	cmd.Stderr = os.Stderr
	l("command: %s", cmd.String())
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	l("output:\n%s", output)

	var hostsObjects []NetworkHost
	lines := strutil.ToSlice(string(output), "\n")

	for _, line := range lines {
		h := checkHostsForm(line)
		id := fmt.Sprintf("network_parse_inventory_host_%s", h.Host)

		if h.User == "" && strings.Contains(h.Host, "@") {
			parts := strings.Split(h.Host, "@")
			h.User = parts[0]
		}

		outrig.Push(id, h)
		hostsObjects = append(hostsObjects, h)
	}
	return hostsObjects, nil
}

func (n *Network) UnmarshalYAML(unmarshal func(interface{}) error) error {
	l := kemba.New("Network.UnmarshalYAML").Printf

	// Temporary struct to hold the unmarshalled data
	type tempNetwork struct {
		Env          EnvList       `yaml:"env"`
		Inventory    string        `yaml:"inventory"`
		Hosts        []NetworkHost `yaml:"hosts"`
		Bastion      string        `yaml:"bastion"`
		User         string        `yaml:"user"`
		IdentityFile string        `yaml:"identity_file"`
	}

	var temp tempNetwork
	if err := unmarshal(&temp); err != nil {
		return err
	}

	// Transfer data from temp struct to the Network struct
	n.Env = temp.Env
	n.Inventory = temp.Inventory
	n.Hosts = temp.Hosts
	n.Bastion = temp.Bastion
	n.User = temp.User
	n.IdentityFile = temp.IdentityFile

	outrig.Push("network_yaml", n)

	l("Unmarshalled network:")
	l("%s", godump.DumpStr(n))
	return nil
}
