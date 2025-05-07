package entity

type ShellCheckFacade interface {
	Check(cmd string, cmdName string) error
	AddNumbers(data []byte) []byte
}
