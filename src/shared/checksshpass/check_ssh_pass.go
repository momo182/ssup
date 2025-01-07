package checksshpass

import (
	"fmt"
	"os"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/strutil"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/lobby"
	"golang.org/x/crypto/ssh"
)

func SetupAuthMethods(authMethods []ssh.AuthMethod, host entity.NetworkHost) []ssh.AuthMethod {
	l := kemba.New("shared::checksshpass::SetupAuthMethods").Println
	password := host.Password

	if *lobby.Lobby.KeyAuth == nil || strutil.IsEmpty(password) {
		fmt.Println("EDF488C4-F467-4279-A031-241F05BCDBC3: no auth methods are set, halting")
		os.Exit(23)
	}

	if !strutil.IsEmpty(password) {
		l("adding password auth to ssh")
		authMethods = []ssh.AuthMethod{
			ssh.Password(password),
			*lobby.Lobby.KeyAuth,
		}
	} else {
		l("not adding password authentication")
		authMethods = []ssh.AuthMethod{
			*lobby.Lobby.KeyAuth,
		}
	}
	l("Auth methods:", authMethods)
	return authMethods
}
