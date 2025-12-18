package appinit

import (
	"strings"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
	"gomodules.xyz/envsubst"
)

func ProcessSupfile(conf *entity.Supfile) error {
	l := kemba.New("uc::appinit::ProcessSupfile").Printf
	commands := &conf.Commands
	envs := conf.Env.GetStore()

	envMap := make(map[string]string)

	// do not add system envs

	// for _, env := range os.Environ() {
	// 	parts := strings.SplitN(env, "=", 2)
	// 	if len(parts) == 2 {
	// 		envMap[parts[0]] = parts[1]
	// 	}
	// }

	// append system envs
	for k, v := range envMap {
		// check if env exists and skip
		if _, ok := envs[k]; !ok {
			envs[k] = v
		}
	}

	for name, command := range commands.Cmds {
		desc := command.Desc

		if desc == "" {
			continue
		}

		if strings.Contains(desc, "$") {
			l("processing evsubst for command desc:")
			l(desc)
			source := desc
			dest, err := envsubst.EvalMap(source, envs)
			if err != nil {
				return err
			}
			l("line after subst:")
			l(dest)
			command.Desc = dest
		}

		commands.Cmds[name] = command
	}

	return nil
}
