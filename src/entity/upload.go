package entity

// UploadOrder represents file copy operation from localhost Src path to Dst
// path of every host in a given Network.
type UploadOrder struct {
	Src string `yaml:"src"`
	Dst string `yaml:"dst"`
	Exc string `yaml:"exclude"`
}

func (u *UploadOrder) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Create a temporary type to avoid infinite recursion
	type rawUploadOrder struct {
		Src         string `yaml:"src"`
		Source      string `yaml:"source"`
		Dst         string `yaml:"dst"`
		Dest        string `yaml:"dest"`
		Destination string `yaml:"destination"`
		Exc         string `yaml:"exclude"`
	}

	var raw rawUploadOrder
	if err := unmarshal(&raw); err != nil {
		return err
	}

	// u.Src = raw.Src
	u.Exc = raw.Exc

	// Check all possible destination fields
	switch {
	case raw.Dst != "":
		u.Dst = raw.Dst
	case raw.Dest != "":
		u.Dst = raw.Dest
	case raw.Destination != "":
		u.Dst = raw.Destination
	}

	switch {
	case raw.Src != "":
		u.Src = raw.Src
	case raw.Source != "":
		u.Src = raw.Source
	}

	return nil
}
