package entity

import (
	"io"
	"os"

	"golang.org/x/crypto/ssh"
)

type ClientFacade interface {
	GetConnection() *ssh.Client
	GetSSHConfig() *ssh.ClientConfig
	Connect(host NetworkHost) error
	SetConnection(*ssh.Client)
	Run(task *Task) error
	Wait() error
	Close() error
	Prefix() (string, int)
	Write(p []byte) (n int, err error)
	WriteClose() error
	Stdin() io.WriteCloser
	Stderr() io.Reader
	Stdout() io.Reader
	Signal(os.Signal) error
	Upload(src string, dest string) error
	Download(src string, dest string, silent bool) error
	GenerateOnRemote(data []byte, dest string) error
	GetHost() string
	GetTube() string
	SetTube(name string)
	GetInventory() *Inventory
	GetShell() string
}

type ArgParserFacade interface {
	Parse(conf *Supfile, initialArgs *InitialArgs, helpMenu HelpDisplayer) (*PlayBook, error)
}
