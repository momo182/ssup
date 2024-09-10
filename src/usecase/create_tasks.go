package usecase

import (
	"io"
	"log"
	"os"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/gateway"
	"github.com/pkg/errors"
)

func CreateTasks(cmd *entity.Command, clients []entity.ClientFacade, env string, args *entity.InitialArgs) ([]*entity.Task, error) {
	l := kemba.New("usecase::create_tasks").Printf
	var tasks []*entity.Task

	// nil guard args
	if args == nil {
		return nil, errors.New("E4DC3F53-A319-4FD5-9A97-FF708A9D3DE7: nil args")
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
		local := &gateway.LocalhostClient{
			Env: env + `export SUP_HOST="localhost";`,
		}
		localHost := entity.NetworkHost{
			Host: "localhost",
		}
		local.Connect(localHost)
		task := &entity.Task{
			Run:     cmd.Local,
			Clients: []entity.ClientFacade{local},
			TTY:     true,
		}
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

	return tasks, nil
}

func openAndReadFile(cmd *entity.Command) ([]byte, error) {
	if cmd == nil {
		log.Panic("F9793EA0-C5C8-43D8-BEE0-4B9AF4EBE7F8: got null command")
	}

	f, err := os.Open(cmd.Script)
	if err != nil {
		return nil, errors.Wrap(err, "can't open script")
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, errors.Wrap(err, "can't read script")
	}

	return data, nil
}

func decorateTaskDetails(args *entity.InitialArgs, task *entity.Task, cmd *entity.Command) {
	if task == nil {
		log.Panic("06878814-005A-427D-B810-1B68739837B1: got null task")
	}

	if cmd == nil {
		log.Panic("47DA54BE-919F-4E40-B7CE-8D8F89CCAE66: got null command")
	}

	if args == nil {
		log.Panic("FB2520DC-4C49-4F43-9A84-C0630EF59579: got null args")
	}

	if args.Debug {
		task.Run = "set -x;" + task.Run
	}
	if cmd.Stdin {
		task.Input = os.Stdin
	}
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
