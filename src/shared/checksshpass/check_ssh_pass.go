package checksshpass

import (
	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/lobby"
	"golang.org/x/crypto/ssh"
)

func CheckPasswordAuth(authMethods []ssh.AuthMethod, host entity.NetworkHost) []ssh.AuthMethod {
	log := kemba.New("sshclient:setup_auth_methods")
	password := host.Password
	if password != "" {
		log.Println("SUDO is set")
		authMethods = []ssh.AuthMethod{
			ssh.Password(password),
			// TODO this key auth may be uninitialized
			*lobby.Lobby.KeyAuth,
		}
	} else {
		log.Println("SUDO not set, not adding password authentication")
		authMethods = []ssh.AuthMethod{
			*lobby.Lobby.KeyAuth,
		}
	}
	log.Println("Auth methods:", authMethods)
	return authMethods
}
