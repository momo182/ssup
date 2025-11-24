package entity

import (
	supfile "github.com/momo182/ssup/src/entity/supfile"
)

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
	Network  *supfile.Network
	Commands []*supfile.Command
}
