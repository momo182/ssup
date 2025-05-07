package entity

import "golang.org/x/crypto/ssh"

type ConnectOrder struct {
	Host         string // will contain ip:port
	ClientConfig *ssh.ClientConfig
}
