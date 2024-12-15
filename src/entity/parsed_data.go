package entity

// ParsedData encapsulates parsed inital arguments
type PlayBook struct {
	plays []Play
}

func (p *PlayBook) AddPlay(play Play) {
	p.plays = append(p.plays, play)
}

func (p *PlayBook) GetPlays() []Play {
	return p.plays
}

type Play struct {
	Nets     *Network
	Commands []*Command
}
