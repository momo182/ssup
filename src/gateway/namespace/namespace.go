package namespace

import (
	"strings"
	"sync"

	"github.com/clok/kemba"
	"github.com/goforj/godump"
	"github.com/momo182/ssup/src/entity"
)

var Namespaces *Namespace

// Namespace stores env information from the session
// it has methods to get and set env variables from any kind of export (bash,sh)
type Namespace struct {
	hostStore map[string]entity.HostNamespace
	mu        sync.RWMutex // Add a mutex
}

func init() {
	Namespaces = New()
}

// InjectNamespacesAndEnvs injects the namespaces and envs into the command
//
// namespaces follow command definitions to overwrite envs with values returned
// from namespaces
func InjectNamespacesAndEnvs(task entity.Task, Env entity.EnvList, c entity.ClientFacade) entity.EnvList {
	l := kemba.New("lobby::InjectNamespacesAndEnvs").Printf

	if len(task.Env.Keys()) > 0 {
		for _, key := range task.Env.Keys() {
			l("injecting task env: %s=%s", key, task.Env.Get(key))

			Env.Set(key, task.Env.Get(key))
		}
	}

	nsEnvs := Namespaces.Get(c.GetHost())
	if len(nsEnvs.EnvStore) > 0 {
		for k, v := range nsEnvs.EnvStore {
			l("injecting namespace env: %s=%s", k, v)
			Env.Set(k, v)
		}
	}

	if c.GetTube() != "" {
		l("found tube: %s, attached to host: %s", c.GetTube(), c.GetHost())
		remoteNs := Namespaces.Get(c.GetTube())
		if len(remoteNs.EnvStore) > 0 {
			for k, v := range remoteNs.EnvStore {
				l("injecting namespace env: %s=%s", k, v)
				Env.Set(k, v)
			}
		}
	}

	l("done injecting namespaces and envs:\n%s", godump.DumpStr(Env))
	return Env
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
	// result.mu is initialized automatically as zero value sync.RWMutex
	return result
}

// Get returns HostNamespace for the given host
func (n *Namespace) Get(host string) entity.HostNamespace {
	l := kemba.New("gateway::Namespace.Get").Printf

	l("check if host has port and drop port")
	host = dropHostPort(host)

	n.mu.RLock()         // Acquire read lock
	defer n.mu.RUnlock() // Ensure read lock is released

	l("check if namespace exists for host: " + host)
	result, ok := n.hostStore[host] // Check if key exists
	if !ok {
		// Handle case where host doesn't exist yet, maybe return empty HostNamespace
		// or consider returning a pointer and a bool (like map lookup)
		// For now, returning the zero value of entity.HostNamespace seems consistent with original logic if key missing.
		// If you want to ensure a HostNamespace is always returned, you might need to create it here,
		// but that would be a write operation and need n.mu.Lock() instead.
		// Let's assume original behavior was to return zero value if not found.
		// l("namespace not found for host: " + host) // Optional log
	}
	l("result: %s", godump.DumpStr(result))
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

	n.mu.Lock()         // Acquire write lock
	defer n.mu.Unlock() // Ensure write lock is released

	n.hostStore[host] = hns
}

// ParseEnvs parses environment variables and adds them to the namespace for the given host.
// Note: This assumes the host namespace already exists.
func (n *Namespace) ParseEnvs(input string, host string) {
	l := kemba.New("gateway::Namespace.ParseEnvs").Printf

	l("check if host has port and drop port")
	host = dropHostPort(host)

	l("parsing envs: %s", input)
	lines := strings.Split(input, "\n")

	n.mu.Lock()         // Acquire write lock as we are modifying EnvStore
	defer n.mu.Unlock() // Ensure write lock is released

	for _, line := range lines {
		l("parsing line: %s", line)

		if line == "" {
			continue // Skip empty lines
		}

		equalsLocation := strings.Index(line, "=")
		// Check if '=' was found
		if equalsLocation == -1 {
			l("skipping line without '=': %s", line)
			continue
		}
		// Ensure there's a key part
		if equalsLocation == 0 {
			l("skipping line with empty key: %s", line)
			continue
		}

		key := line[:equalsLocation]
		value := strings.TrimSpace(line[equalsLocation+1:])
		l("key: %s, value: %s", key, value)

		// Ensure host entry exists before trying to write to its EnvStore
		// This mimics the pattern in SetFromEnvString more closely.
		// If you are sure ParseEnvs is only called after Add, you might skip this check.
		if _, exists := n.hostStore[host]; !exists {
			l("host namespace does not exist, creating for host: %s", host)
			n.hostStore[host] = NewHostNamespace()
		}

		// Now it's safe to write (assuming EnvStore was initialized in NewHostNamespace)
		n.hostStore[host].EnvStore[key] = value
	}
}

// func (n *Namespace) SetFromEnvString(input string, host string) {
// 	l := kemba.New("gateway::Namespace::SetFromEnvString").Printf
// 	customNamespace := ""

// 	l("check if host has port and drop port")
// 	host = dropHostPort(host)

// 	l("host: %s", host)
// 	l("setting envs from string:\n%s", input)
// 	lines := strings.Split(input, "\n")
// 	for _, line := range lines {
// 		l("reading line: %s", line)
// 		if line == "" {
// 			l("skip empty line")
// 			continue
// 		}

// 		if !strings.Contains(line, "=") {
// 			l("failed to find = inside line: %s", line)
// 			continue
// 		}

// 		// split on " " and count parts
// 		parts := strings.Split(line, " ")
// 		l("parts: %s", godump.DumpStr(parts))
// 		switch len(parts) {
// 		case 1:
// 			l("setting env for host: %s", host)
// 		case 2:
// 			l("tube found, setting env for tube: %s", parts[0])
// 			line = parts[1]
// 			host = parts[0]
// 		default:
// 			l("ERROR: failed to parse line: '%s'", line)
// 			return
// 		}

// 		equalsLocation := strings.Index(line, "=")
// 		// check we've found any = at all?
// 		if equalsLocation == -1 {
// 			l("failed to find = inside line: %s", line)
// 			continue
// 		}

// 		key := line[:equalsLocation]
// 		value := strings.TrimSpace(line[equalsLocation+1:])
// 		l("key: %s, value: %s", key, value)

// 		// check if key has a space in it
// 		if strings.Contains(key, " ") {
// 			l("key contains space, splitting")
// 			customNamespace = strings.Split(key, " ")[0]
// 			l("customNamespace: %s", customNamespace)
// 		}

// 		if customNamespace != "" {
// 			l("setting env for custom namespace: %s", customNamespace)
// 			host = customNamespace
// 		}

// 		l("host store: %s", godump.DumpStr(n))
// 		if n.hostStore == nil {
// 			l("host store is nil, better create new, should neve happen")
// 		}

// 		if n.hostStore[host].EnvStore == nil {
// 			l("host store is nil, creating new namespace: %s", host)
// 			n.hostStore[host] = NewHostNamespace()
// 		}

// 		n.hostStore[host].EnvStore[key] = value
// 	}
// }

// SetFromEnvString sets environment variables from a string input, potentially for different hosts/namespaces.
func (n *Namespace) SetFromEnvString(input string, defaultHost string) {
	l := kemba.New("gateway::Namespace::SetFromEnvString").Printf
	customNamespace := ""

	l("check if host has port and drop port for defaultHost")
	defaultHost = dropHostPort(defaultHost)

	l("defaultHost: %s", defaultHost)
	l("setting envs from string:\n%s", input)
	lines := strings.Split(input, "\n")

	n.mu.Lock()         // Acquire write lock as we are modifying hostStore and EnvStores
	defer n.mu.Unlock() // Ensure write lock is released

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
		l("parts: %s", godump.DumpStr(parts))
		hostForThisLine := defaultHost // Start with the default host
		envLine := line                // Start with the full line as the env definition

		switch len(parts) {
		case 1:
			l("setting env for default host: %s", hostForThisLine)
			// hostForThisLine remains defaultHost, envLine remains line
		case 2:
			l("tube/namespace found, setting env for: %s", parts[0])
			hostForThisLine = parts[0] // Override host
			envLine = parts[1]         // Override line to parse
		default:
			l("ERROR: failed to parse line (too many parts): '%s'", line)
			// Consider if you should return or just skip this line
			continue // Skip this malformed line
		}

		// Now parse the envLine (which is either the original line or the part after the host)
		equalsLocation := strings.Index(envLine, "=")
		// check we've found any = at all?
		if equalsLocation == -1 || equalsLocation == 0 { // Check for missing or empty key
			l("failed to find valid = inside env definition part: %s", envLine)
			continue
		}

		key := envLine[:equalsLocation]
		value := strings.TrimSpace(envLine[equalsLocation+1:])
		l("parsed key: %s, value: %s for host: %s", key, value, hostForThisLine)

		// check if key has a space in it (this logic seems slightly off - it checks *key* again)
		// Perhaps it was meant to check the original line or parts?
		// Based on original code, it checks *key* again.
		if strings.Contains(key, " ") {
			l("key contains space, splitting key (potentially problematic logic)")
			// This will take the first part of the key if it contains a space
			// This seems like unusual logic for an env var key. Verify if intended.
			partsOfKey := strings.Split(key, " ")
			if len(partsOfKey) > 0 {
				customNamespace = partsOfKey[0] // Take the first part as custom namespace name
				l("customNamespace derived from key: %s", customNamespace)
				// Override host again?
				hostForThisLine = customNamespace
			}
		}

		// --- Critical Section (protected by n.mu.Lock() above) ---
		// Ensure the host namespace exists in hostStore
		if _, exists := n.hostStore[hostForThisLine]; !exists {
			l("host store entry missing, creating new namespace for host: %s", hostForThisLine)
			n.hostStore[hostForThisLine] = NewHostNamespace()
		}

		// Perform the actual write to the map
		n.hostStore[hostForThisLine].EnvStore[key] = value
		// --- End Critical Section ---
	}
}
