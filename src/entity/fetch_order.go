package entity

type FetchOrder struct {
	Host string `yaml:"host"`
	Src  string `yaml:"src"`
	Dst  string `yaml:"dst"`
}
