package lobby

import (
	"github.com/momo182/ssup/src/gateway/shellcheck"
	"golang.org/x/crypto/ssh"
)

type ServiceLobby struct {
	KeyAuth    *ssh.AuthMethod
	Shellcheck *shellcheck.ShellCheck
}

var Lobby *ServiceLobby
