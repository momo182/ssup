package entity

// Upload represents file copy operation from localhost Src path to Dst
// path of every host in a given Network.
type Upload struct {
	Src string `yaml:"src"`
	Dst string `yaml:"dst"`
	Exc string `yaml:"exclude"`
}
