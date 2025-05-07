package namespace

import (
	"strings"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/dump"
	"github.com/momo182/ssup/src/entity"
)

// Namespace stores env information from the session
// it has methods to get and set env variables from any kind of export (bash,sh)
type Namespace struct {
	hostStore map[string]entity.HostNamespace
}

func NewHostNamespace() entity.HostNamespace {
	l := kemba.New("gateway::namespace::Namespace::NewHostNamespace").Printf
	l("creating new namespace")

	hns := entity.HostNamespace{}
	store := make(map[string]string)
	hns.EnvStore = store

	return hns

}

func New() *Namespace {
	result := new(Namespace)
	store := make(map[string]entity.HostNamespace)
	result.hostStore = store
	return result
}

// Get returns HostNamespace for the given host
func (n *Namespace) Get(host string) entity.HostNamespace {
	l := kemba.New("gateway::Namespace.Get").Printf

	l("check if host has port and drop port")
	host = dropHostPort(host)

	l("check if namespace exists for host: " + host)
	result := n.hostStore[host]
	l("result: %s", dump.Format(result))
	return result
}

func dropHostPort(host string) string {
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}
	return host
}

// add namespace for host
func (n *Namespace) Add(host string) {
	l := kemba.New("gateway::Namespace.Add").Printf

	l("check if host has port and drop port")
	host = dropHostPort(host)

	l("add namespace for host: " + host)
	hns := entity.HostNamespace{}
	store := make(map[string]string)
	hns.EnvStore = store
	n.hostStore[host] = hns
}

func (n *Namespace) ParseEnvs(input string, host string) {
	l := kemba.New("gateway::Namespace.ParseEnvs").Printf

	l("check if host has port and drop port")
	host = dropHostPort(host)

	l("parsing envs: %s", input)
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		l("parsing line: %s", line)

		equalsLocation := strings.Index(line, "=")
		key := line[:equalsLocation]
		value := strings.TrimSpace(line[equalsLocation+1:])
		l("key: %s, value: %s", key, value)
		n.hostStore[host].EnvStore[key] = value
	}
}

func (n *Namespace) SetFromEnvString(input string, host string) {
	l := kemba.New("gateway::Namespace::SetFromEnvString").Printf
	customNamespace := ""

	l("check if host has port and drop port")
	host = dropHostPort(host)

	l("host: %s", host)
	l("setting envs from string:\n%s", input)
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		l("reading line: %s", line)
		if line == "" {
			l("skip empty line")
			continue
		}

		if !strings.Contains(line, "=") {
			l("failed to find = inside line: %s", line)
			continue
		}

		// split on " " and count parts
		parts := strings.Split(line, " ")
		l("parts: %s", dump.Format(parts))
		switch len(parts) {
		case 1:
			l("setting env for host: %s", host)
		case 2:
			l("tube found, setting env for tube: %s", parts[0])
			line = parts[1]
			host = parts[0]
		default:
			l("ERROR: failed to parse line: '%s'", line)
			return
		}

		equalsLocation := strings.Index(line, "=")
		// check we've found any = at all?
		if equalsLocation == -1 {
			l("failed to find = inside line: %s", line)
			continue
		}

		key := line[:equalsLocation]
		value := strings.TrimSpace(line[equalsLocation+1:])
		l("key: %s, value: %s", key, value)

		// check if key has a space in it
		if strings.Contains(key, " ") {
			l("key contains space, splitting")
			customNamespace = strings.Split(key, " ")[0]
			l("customNamespace: %s", customNamespace)
		}

		if customNamespace != "" {
			l("setting env for custom namespace: %s", customNamespace)
			host = customNamespace
		}

		l("host store: %s", dump.Format(n))
		if n.hostStore == nil {
			l("host store is nil, better create new, should neve happen")
		}

		if n.hostStore[host].EnvStore == nil {
			l("host store is nil, creating new namespace: %s", host)
			n.hostStore[host] = NewHostNamespace()
		}

		n.hostStore[host].EnvStore[key] = value
	}
}
