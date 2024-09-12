package entity

import (
	"os"
	"os/exec"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/strutil"
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

	// cmdParts := strings.Fields()
	// if len(cmdParts) == 0 {
	// 	l("no inventory given")
	// 	return nil, nil

	// }

	cmdParts := []string{"/bin/sh", "-c", n.Inventory}
	l("%v", cmdParts)
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, n.Env.Slice()...)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	l("output:\n%s", output)

	// var hosts []string
	var hostsObjects []NetworkHost
	// buf := bytes.NewBuffer(output)
	lines := strutil.ToSlice(string(output), "\n")

	for _, line := range lines {
		// host, err := buf.ReadString('\n')
		// if err != nil {
		// 	if err == io.EOF {
		// 		break
		// 	}
		// 	return nil, err
		// }
		h := checkHostsForm(line)

		// 	host = strings.TrimSpace(host)
		// 	// skip empty lines and comments
		// 	if host == "" || host[:1] == "#" {
		// 		continue
		// 	}

		// 	hosts = append(hosts, host)
		// }
		// for _, host := range hosts {
		// 	h := NetworkHost{
		// 		Host: host,
		// 	}

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

	l("Unmarshalled network: %+v", *n)
	return nil
}