package entity

import (
	"strings"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/strutil"
)

// Command represents command(s) to be run remotely.
type Command struct {
	Name     string      `yaml:"-"`      // Command name.
	Desc     string      `yaml:"desc"`   // Command description.
	Local    string      `yaml:"local"`  // Command(s) to be run locally.
	Run      string      `yaml:"run"`    // Command(s) to be run remotelly.
	Script   string      `yaml:"script"` // Load command(s) from script and run it remotelly.
	Upload   []*Upload   `yaml:"upload"` // See Upload struct.
	Copy     *CopyOrder  `yaml:"copy"`   // See Upload struct.
	Stdin    bool        `yaml:"stdin"`  // Attach localhost STDOUT to remote commands' STDIN?
	Once     bool        `yaml:"once"`   // The command should be run "once" (on one host only).
	Serial   int         `yaml:"serial"` // Max number of clients processing a task in parallel.
	Fetch    *FetchOrder `yaml:"fetch" ` // See Fetch struct.
	Sudo     bool        `yaml:"sudo" `  // Run command(s) as root?
	SudoPass string      `yaml:"sudo_pass"`

	// API backward compatibility. Will be deprecated in v1.0.
	RunOnce bool `yaml:"run_once"` // The command should be run once only.
}

// Commands is a list of user-defined commands
type Commands struct {
	Names []string
	Cmds  map[string]Command
}

// func (n *Networks) UnmarshalYAML(unmarshal func(interface{}) error) error {
// 	l := kemba.New("Networks.UnmarshalYAML").Printf

// 	// Temporary map to unmarshal the YAML data
// 	temp := make(map[string]*Network)
// 	if err := unmarshal(&temp); err != nil {
// 		return err
// 	}

// 	// Initialize the Nets map and Names slice
// 	n.Nets = make(map[string]*Network)
// 	n.Names = make([]string, len(temp))

// 	// Populate the Nets map and Names slice
// 	i := 0
// 	for key, value := range temp {
// 		n.Names[i] = key
// 		n.Nets[key] = value
// 		i++
// 	}

// 	spew.Config.MaxDepth = 6
// 	l("dumps:\nfinal:\n%s\nsource:\n%s", spew.Sdump(n.Nets), spew.Sdump(temp))
// 	spew.Config.MaxDepth = 3

// 	return nil
// }

func (c *Commands) UnmarshalYAML(unmarshal func(interface{}) error) error {
	l := kemba.New("commands.unmarshal_yaml").Printf
	temp := make(map[string]Command)

	l("unmarshal to temp")
	if err := unmarshal(&temp); err != nil {
		l("F6A7812D-4073-43BA-B90D-2B1E07CB304E: failed to unmarshal to temp struct (%s)", err.Error())
		return err
	}

	c.Names = make([]string, len(temp))
	c.Cmds = make(map[string]Command)

	l("got items")
	i := 0
	for key, value := range temp {
		l("got item")
		c.Names[i] = key
		value.Name = key
		rawCommand := value.Run
		value.Run = processSourceLinks(rawCommand)
		c.Cmds[key] = value

		// type Command struct {
		//     Name   string      `yaml:"-"`      // Command name.
		//     Desc   string      `yaml:"desc"`   // Command description.
		//     Local  string      `yaml:"local"`  // Command(s) to be run locally.
		//     Run    string      `yaml:"run"`    // Command(s) to be run remotelly.
		//     Script string      `yaml:"script"` // Load command(s) from script and run it remotelly.
		//     Upload []Upload    `yaml:"upload"` // See Upload struct.
		//     Copy   *CopyOrder  `yaml:"copy"`   // See Upload struct.
		//     Stdin  bool        `yaml:"stdin"`  // Attach localhost STDOUT to remote commands' STDIN?
		//     Once   bool        `yaml:"once"`   // The command should be run "once" (on one host only).
		//     Serial int         `yaml:"serial"` // Max number of clients processing a task in parallel.
		//     Fetch  *FetchOrder `yaml:"fetch" ` // See Fetch struct.
		//     Sudo   bool        `yaml:"sudo" `  // Run command(s) as root?

		//     // API backward compatibility. Will be deprecated in v1.0.
		//     RunOnce bool `yaml:"run_once"` // The command should be run once only.
		// }

		l("item key: %s", key)
		l("item value:\n---------------\nname: %s\ndesc: %s\nlocal: %v\nrun:\n%s\nscript: %s\nupload: %v\ncopy: %v\nfetch: %v\nsudo: %v\n---------------", value.Name, value.Desc, value.Local, value.Run, value.Script, value.Upload, value.Copy, value.Fetch, value.Sudo)
		i++
	}
	return nil
}

// processSourceLinks will scanthrough lines and replace
// the `#source://` links with the actual content.
func processSourceLinks(rawCommand string) string {
	l := kemba.New("entity -> process_source_links").Printf
	rawLines := strings.Split(rawCommand, "\n")
	result := []byte{}
	count := 0

	for _, line := range rawLines {
		if strings.HasPrefix(line, "#source://") {
			l("matched line: %s", line)
			file := strings.TrimPrefix(line, "#source://")
			if strings.Contains(file, "#") {
				// must have a # comment behind the value
				// will have to strip the # comment
				// by searching for the # comment location and
				// trimming the length of the value
				location := strings.Index(file, "#")
				file = file[:location]
			}
			data := fsutil.ReadString(file)
			doSkip := false

			// drop shebangs
			fl := strutil.FirstLine(data)
			if (strings.HasPrefix(fl, "#!/")) && (count == 0) {
				l("$$$$$$$$$$$$$$$$$$$$$$$$$$$$ shebang found $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$")
				doSkip = true
			}
			l("before:\n%s", "'"+strutil.FirstLine(data)+"'")

			index := strings.Index(data, "\n")
			if doSkip {
				data = data[index+1:]
			}
			l("after:\n%s", "'"+strutil.FirstLine(data)+"'")

			result = append(result, []byte(data+"\n")...)
			continue
		}
		result = append(result, []byte(line+"\n")...)
	}
	return string(result)
}

func (c *Commands) Get(name string) (Command, bool) {
	cmd, ok := c.Cmds[name]
	return cmd, ok
}