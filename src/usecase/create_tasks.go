package usecase

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/dump"
	"github.com/momo182/ssup/src/entity"

	"github.com/momo182/ssup/src/gateway/localhost"
	"github.com/pkg/errors"
	"github.com/samber/oops"
)

func CreateTasks(cmd *entity.Command, clients []entity.ClientFacade, env entity.EnvList, args *entity.InitialArgs) ([]*entity.Task, error) {
	l := kemba.New("usecase::create_tasks").Printf
	var tasks []*entity.Task

	// nil guard args
	if args == nil {
		return nil, errors.New("E4DC3F53-A319-4FD5-9A97-FF708A9D3DE7: nil args")
	}

	// nil guard cmd
	if cmd == nil {
		return nil, errors.New("BCD9255E-BB43-400A-92AB-28E70B07F5D2: nil cmd")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "resolving CWD failed")
	}
	l("cwd: " + cwd)

	// TODO remove this, source:// function mirrors this
	// Script. Read the file as a multiline input command.
	l("check if script")
	if cmd.Script != "" {
		data, err := openAndReadFile(cmd)
		if err != nil {
			return nil, err
		}

		task := entity.Task{
			Run:  string(data),
			TTY:  true,
			Sudo: cmd.Sudo,
		}

		decorateTaskDetails(args, &task, cmd)

		if cmd.Once {
			task.Clients = []entity.ClientFacade{clients[0]}
			tasks = append(tasks, &task)
		} else if cmd.Serial > 0 {
			// Each "serial" task client group is executed sequentially.
			tasks = processClientsInGroups(clients, cmd, task, tasks)
		} else {
			task.Clients = clients
			tasks = append(tasks, &task)
		}
	}

	// Local command.
	l("check if local command")
	if cmd.Local != "" {
		envStore := new(entity.EnvList)
		envStore.Set("SUP_HOST", "localhost")
		local := &localhost.LocalhostClient{
			Env: envStore,
		}
		localHost := entity.NetworkHost{
			Host: "localhost",
		}
		local.Connect(localHost)
		task := &entity.Task{
			Run:     cmd.Local,
			Clients: []entity.ClientFacade{local},
			TTY:     true,
			Env:     env,
		}

		AppendCommandEnvsToTask(cmd, task)
		decorateTaskDetails(args, task, cmd)
		tasks = append(tasks, task)
	}

	// Remote command.
	l("check if remote command")
	if cmd.Run != "" {
		task := entity.Task{
			Run:  cmd.Run,
			TTY:  true,
			Sudo: cmd.Sudo,
			Env:  env,
		}

		AppendCommandEnvsToTask(cmd, &task)
		decorateTaskDetails(args, &task, cmd)

		if cmd.Once {
			task.Clients = []entity.ClientFacade{clients[0]}
			tasks = append(tasks, &task)
		} else if cmd.Serial > 0 {
			// Each "serial" task client group is executed sequentially.
			tasks = processClientsInGroups(clients, cmd, task, tasks)
		} else {
			task.Clients = clients
			tasks = append(tasks, &task)
		}
	}

	return tasks, nil
}

func AppendCommandEnvsToTask(cmd *entity.Command, task *entity.Task) {
	l := kemba.New("usecase::append_command_envs_to_task").Printf

	l("dump: 61B6EDEA-0EA0-4466-95CB-08AEEB7D0258")
	l("before:")
	l(dump.Format(task))

	if cmd == nil {
		log.Panic("20ABE4E4-447D-443A-A3F2-36A9E61C721C: got null command")
	}

	if task == nil {
		log.Panic("36A27F8A-1A21-4CA3-9A42-3B1A379B3436: got null task")
	}

	l("check if command had envs")

	if len(cmd.Env.Keys()) == 0 {
		l("command had no env, skipping")
		return
	}

	l("command had some env, applying envs to task")
	// task.Env = append(task.Env, cmd.Env...)
	source := cmd.Env
	// dest := task.Env
	// dest := entity.EnvList{}
	// append every task from source to dest
	for _, key := range source.Keys() {
		value := source.Get(key)
		l("got value: %s", value)
		task.Env.Set(key, value)
	}

	l("dump: 65589739-F0F5-4D96-A535-07684F4F5CC0")
	l("after:")
	l(dump.Format(task))

	l("done appending command envs")
}

func openAndReadFile(cmd *entity.Command) ([]byte, error) {
	if cmd == nil {
		log.Panic("F9793EA0-C5C8-43D8-BEE0-4B9AF4EBE7F8: got null command")
	}

	f, e := os.Open(cmd.Script)
	if e != nil {
		return nil, oops.Trace("8FCD95B1-9A22-4E10-807C-E37EE72981AE").
			Hint("openining file").
			With("file", cmd.Script).
			Wrap(e)
	}

	data, e := io.ReadAll(f)
	if e != nil {
		return nil, oops.Trace("EDCD973B-85E6-48B1-BC69-3C0BDD0B8C73").
			Hint("reading script").
			With("file", f).
			Wrap(e)
	}

	return data, nil
}

func decorateTaskDetails(args *entity.InitialArgs, task *entity.Task, cmd *entity.Command) {
	l := kemba.New("usecase::decorate_task_details").Printf
	if task == nil {
		fmt.Println("06878814-005A-427D-B810-1B68739837B1: got null task")
	}

	if cmd == nil {
		fmt.Println("47DA54BE-919F-4E40-B7CE-8D8F89CCAE66: got null command")
	}

	if args == nil {
		fmt.Println("FB2520DC-4C49-4F43-9A84-C0630EF59579: got null args")
	}

	if args.Debug {
		task.Run = "set -x;" + task.Run
	}

	if cmd.Stdin {
		l("adding stdin as input to task")
		task.Input = os.Stdin
	}
	l("dump: 35C3F8BB-1FAC-4C47-B996-C513AF335CA7")
	l("task about to run here:")
	l(dump.Format(task))
}

func processClientsInGroups(clients []entity.ClientFacade, cmd *entity.Command, task entity.Task, tasks []*entity.Task) []*entity.Task {
	if cmd == nil {
		log.Panic("908452A6-8978-4B42-97B1-854673457551: got null command")
	}

	if tasks == nil {
		log.Panic("26858814-005A-427D-B810-1B68739837B1: got null tasks")
	}

	for i := 0; i < len(clients); i += cmd.Serial {
		j := i + cmd.Serial
		if j > len(clients) {
			j = len(clients)
		}
		copy := task
		copy.Clients = clients[i:j]
		tasks = append(tasks, &copy)
	}

	return tasks
}
