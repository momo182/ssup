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
	port      int
}

func NewHostNamespace() entity.HostNamespace {
	l := kemba.New("gateway::namespace::Namespace.NewHostNamespace").Printf
	l("creating new namespace")

	hns := entity.HostNamespace{}
	store := make(map[string]string)
	hns.EnvStore = store

	return hns

}

func New() Namespace {
	store := make(map[string]entity.HostNamespace)
	return Namespace{
		hostStore: store,
	}
}

// Get returns HostNamespace for the given host
func (n *Namespace) Get(host string) entity.HostNamespace {
	l := kemba.New("gateway::Namespace.Get").Printf
	l("check if namespace exists for host: " + host)
	result := n.hostStore[host]
	l("result: %s", dump.Format(result))
	return result
}

// add namespace for host
func (n *Namespace) Add(host string) {
	l := kemba.New("gateway::Namespace.Add").Printf
	l("add namespace for host: " + host)
	n.hostStore[host] = NewHostNamespace()
}

func (n *Namespace) ParseEnvs(input string, host string) {
	l := kemba.New("gateway::Namespace.ParseEnvs").Printf
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
