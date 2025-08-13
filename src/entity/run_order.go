package entity

type RunOrder struct {
	MakefileMode bool
	Network      *Network
	Vars         EnvList
	Commands     []*Command
	Args         InitialArgs
}
