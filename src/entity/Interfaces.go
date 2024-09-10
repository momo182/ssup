package entity

import (
	"io"
	"os"
)

type ClientFacade interface {
	Connect(host NetworkHost) error
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
}
