package supfile

type DownloadOrder struct {
	Host string `yaml:"host"`
	Src  string `yaml:"src"`
	Dst  string `yaml:"dst"`
	Flat bool
}
