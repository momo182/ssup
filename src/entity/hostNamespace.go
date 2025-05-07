package entity

// HostNamespace stores env vars from the host
// it has methods to get and set env variables
type HostNamespace struct {
	EnvStore map[string]string
}

// Get returns the value of the env var
func (h *HostNamespace) Get(key string) string {
	return h.EnvStore[key]
}

// Set sets the value of the env var
func (h HostNamespace) Set(key, value string) {
	h.EnvStore[key] = value
}

// Unset unsets the value of the env var
func (h *HostNamespace) Unset(key string) {
	delete(h.EnvStore, key)
}
