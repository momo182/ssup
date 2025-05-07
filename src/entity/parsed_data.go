package entity

type PlayBook struct {
	plays      []Play
	isMakefile bool
}

func (p *PlayBook) AddPlay(play Play) {
	p.plays = append(p.plays, play)
}

func (p *PlayBook) GetPlays() []Play {
	return p.plays
}

func (p *PlayBook) MarkAsMakefileMode() {
	p.isMakefile = true
}

func (p *PlayBook) IsMakefileMode() bool {
	return p.isMakefile
}

type Play struct {
	Network  *Network
	Commands []*Command
}
