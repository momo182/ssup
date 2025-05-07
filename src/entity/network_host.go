package entity

import (
	"github.com/clok/kemba"
	"github.com/davecgh/go-spew/spew"
	"github.com/gookit/goutil/dump"
	"github.com/samber/oops"
)

type NetworkHost struct {
	Host     string  `yaml:"host"`
	User     string  `yaml:"user"`
	Password string  `yaml:"pass"`
	Tube     string  `yaml:"tube"`
	Env      EnvList `yaml:"env"`
	Sudo     bool    `yaml:"sudo" default:"false"`
	// Namespace string  `yaml:"namespace" default:""`
}

func (n *NetworkHost) UnmarshalYAML(unmarshal func(interface{}) error) error {
	l := kemba.New("NetworkHost.UnmarshalYAML").Printf

	// First, attempt to unmarshal as a plain string
	var hostString string
	if err := unmarshal(&hostString); err == nil {
		l("+++ host as string found: %s", hostString)
		*n = checkHostsForm(hostString)
		return nil
	}

	// If it fails, attempt to unmarshal as an object
	type tempNetworkHost struct {
		User     string  `yaml:"user"`
		Password string  `yaml:"pass"`
		Tube     string  `yaml:"tube"`
		Host     string  `yaml:"host"`
		Env      EnvList `yaml:"env"`
		Sudo     bool    `yaml:"sudo" default:"false"`
	}

	var temp tempNetworkHost
	if err := unmarshal(&temp); err != nil {
		return err
	}

	// check if password is shell resolve case
	if IsShell(temp.Password) {
		l("!!! shell resolve case found")
		var e error
		temp.Password, e = ResolveShell(temp.Password)
		if e != nil {
			return e
		}
	}

	l("!!! host as object found: %s", temp.Password)
	n.Host = temp.Host
	n.User = temp.User
	n.Password = temp.Password
	n.Tube = temp.Tube
	n.Env = temp.Env

	return nil
}

func checkHostsForm(host string) NetworkHost {
	var result NetworkHost
	l := kemba.New("CheckHostsForm").Printf

	l("host as string: %s", host)

	// value of -1 in start positions will indicate
	// no values for given args/fields
	passwordStart := findPasswordStart(host)
	tubeNameStart := findTubeNameStart(host)
	passwordEnd := findPasswordEnd(host)
	tubeNameEnd := findTubeNameEnd(host)
	newHost := ""
	password := ""
	tube := ""
	positions := map[string]int{
		"passwordStart": passwordStart,
		"tubeNameStart": tubeNameStart,
		"passwordEnd":   passwordEnd,
		"tubeNameEnd":   tubeNameEnd,
	}
	l("dump: DE462638-4225-44C6-852F-4F20AEEC2A0D")
	l("%s", dump.Format(positions))

	if passwordStart < 0 && tubeNameStart < 0 {
		l("no pass and tube found")
		// no need to do anything
		newHost = host
	}

	if passwordStart > 0 {
		l("password found")
		// both fields set, we need password start and tube name start
		// to get back the value where password ends

		// everything after password start separator is fields
		newHost = host[:passwordStart-len(PassSeparator)]

		// check if tube name start is there or end of the host is the end of the password
		if tubeNameStart > 0 {
			l("password > tube name found")
			passwordEnd = tubeNameStart - len(TubeNameSeparator)
		} else {
			l("password > no tube")
			passwordEnd = len(host)

		}

		l("done checking pass")
		password = host[passwordStart:passwordEnd]
	}

	if tubeNameStart > 0 {
		l("tube found")
		// simple case if start is found end is always len f the host
		// but first check if newHost was set by password check
		if newHost == "" {
			newHost = host[:tubeNameStart-len(TubeNameSeparator)]
		}
		tube = host[tubeNameStart:tubeNameEnd]
	}

	result.Host = newHost
	result.Password = password
	result.Tube = tube

	// check if password is shell resolve case
	if IsShell(result.Password) {
		pass, e := ResolveShell(result.Password)
		if e != nil {
			l(oops.Trace("765212F7-64B0-4974-9A50-E8B8C1807FFE").
				Hint("resolving password via shell").
				With("result.Password", result.Password).
				Wrap(e).Error())
		}
		result.Password = pass
	}

	spew.Config.MaxDepth = 5
	l(dump.Format("dump: 3DB74440-E5D9-4BEE-89D8-9C4EEB1459A9", result))
	spew.Config.MaxDepth = SPEW_DEPTH
	l("finished checking nets")
	return result
}
