package entity

import (
	"io"
	"os"

	"github.com/momo182/ssup/src/entity/supfile"
	"golang.org/x/crypto/ssh"
)

type ClientFacade interface {
	GetConnection() *ssh.Client
	GetSSHConfig() *ssh.ClientConfig
	Connect(host supfile.NetworkHost) error
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
	Download(src string, dest string, silent bool, flat bool) error
	GenerateOnRemote(data []byte, dest string) error
	GetHost() string
	GetTube() string
	SetTube(name string)
	GetInventory() *Inventory
	GetShell() string
	String() string
	CopyDistFolders() error
}

type ArgParserFacade interface {
	Parse(conf *supfile.Supfile, initialArgs *InitialArgs, helpMenu HelpDisplayer) (*PlayBook, error)
}
