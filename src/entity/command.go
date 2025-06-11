package entity

import (
	"strings"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/strutil"
	"gopkg.in/yaml.v2"
)

// Command represents command(s) to be run remotely.
type Command struct {
	Name   string    `yaml:"-"`      // Command name.
	Desc   string    `yaml:"desc"`   // Command description.
	Local  string    `yaml:"local"`  // Command(s) to be run locally.
	Run    string    `yaml:"run"`    // Command(s) to be run remotelly.
	Script string    `yaml:"script"` // Load command(s) from script and run it remotelly.
	Upload []*Upload `yaml:"upload"` // See Upload struct.
	// Copy     *CopyOrder  `yaml:"copy"`   // See Upload struct.
	Stdin    bool        `yaml:"stdin"`  // Attach localhost STDOUT to remote commands' STDIN?
	Once     bool        `yaml:"once"`   // The command should be run "once" (on one host only).
	Serial   int         `yaml:"serial"` // Max number of clients processing a task in parallel.
	Fetch    *FetchOrder `yaml:"fetch" ` // See Fetch struct.
	Sudo     bool        `yaml:"sudo" `  // Run command(s) as root?
	SudoPass string      `yaml:"sudo_pass"`
	Env      EnvList     `yaml:"env"`

	// API backward compatibility. Will be deprecated in v1.0.
	RunOnce bool `yaml:"run_once"` // The command should be run once only.
}

// Commands is a list of user-defined commands
type Commands struct {
	Names []string
	Cmds  map[string]Command
}

func (c *Commands) UnmarshalYAML(unmarshal func(interface{}) error) error {
	l := kemba.New("entity::Commands.UnmarshalYAML").Printf
	var raw yaml.MapSlice

	l("unmarshal to raw yaml.MapSlice")
	if err := unmarshal(&raw); err != nil {
		return err
	}

	c.Cmds = make(map[string]Command)
	c.Names = make([]string, 0, len(raw))

	for _, item := range raw {
		key, ok := item.Key.(string)
		if !ok {
			continue // or return error
		}
		valBytes, err := yaml.Marshal(item.Value)
		if err != nil {
			return err
		}
		var cmd Command
		if err := yaml.Unmarshal(valBytes, &cmd); err != nil {
			return err
		}

		cmd.Name = key
		cmd.Run = processSourceLinks(cmd.Run)

		c.Names = append(c.Names, key)
		c.Cmds[key] = cmd

		l("item key: %s", key)
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
		if strings.HasPrefix(line, SourceDirective) {
			l("matched line: %s", line)
			file := strings.TrimPrefix(line, SourceDirective)
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

func (c *Commands) Has(name string) bool {
	_, ok := c.Cmds[name]
	return ok
}
