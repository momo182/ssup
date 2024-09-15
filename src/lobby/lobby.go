package lobby

import (
	"github.com/momo182/ssup/src/gateway/namespace"
	"github.com/momo182/ssup/src/gateway/shellcheck"
	"golang.org/x/crypto/ssh"
)

// ServiceLobby
type ServiceLobby struct {
	KeyAuth    *ssh.AuthMethod
	Shellcheck *shellcheck.ShellCheck
	Namespaces namespace.Namespace
}

// Lobby holds common serices used by many places in code
var Lobby *ServiceLobby

// RegisterCmd is the shell function literal to register a key and value in a file
// later to be parsed by ssup for any envs passed inside
// moved here to reduce code duplication
var RegisterCmd = `register() {
local key=$1
local val=$2
local dest="$HOME/_ssup_vars.env"

if [ -n "$SUDO_USER" ]; then
	# shellcheck disable=SC2116
	local rem_root=$(eval echo "~${SUDO_USER}")
	local dest="$rem_root/_ssup_vars.env"
fi

echo "${key}=${val}" >> "$dest"
}
`
