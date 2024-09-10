package usecase

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/goware/prefixer"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/gateway"
	svc "github.com/momo182/ssup/src/lobby"
	"github.com/pkg/errors"
	"github.com/samber/oops"
	"golang.org/x/crypto/ssh"
)

type Stackup struct {
	conf   *entity.Supfile
	debug  bool
	prefix bool
	Args   *entity.InitialArgs
}

func NewStackup(conf *entity.Supfile) (*Stackup, error) {
	return &Stackup{
		conf: conf,
	}, nil
}

// Run runs set of commands on multiple hosts defined by network sequentially.
// TODO: This megamoth method needs a big refactor and should be split
//
//	to multiple smaller methods.
func (sup *Stackup) Run(network *entity.Network, envVars entity.EnvList, commands ...*entity.Command) error {
	if len(commands) == 0 {
		return oops.Trace("1DF987CA-1E81-4C48-9163-DDAEA0C0CDF7").
			Hint("check how many commands").
			Wrap(errors.New("no commands to be run"))
	}

	env := envVars.AsExport()

	// Create clients for every host (either SSH or Localhost).
	bastion, e := sup.connectToBationHost(network)
	if e != nil {
		return oops.
			Trace("60267BB8-0049-4819-BA06-30EFDFE161AD").
			Wrap(e)
	}

	var wg sync.WaitGroup
	clientCh := make(chan entity.ClientFacade, len(network.Hosts))
	errCh := make(chan error, len(network.Hosts))

	connectToHosts(network, &wg, env, errCh, clientCh, bastion)
	wg.Wait()
	close(clientCh)
	close(errCh)

	maxLen := 0
	var clients []entity.ClientFacade = make([]entity.ClientFacade, 0)
	for client := range clientCh {
		if remote, ok := client.(*gateway.SSHClient); ok {
			defer remote.Close()
		}
		_, prefixLen := client.Prefix()
		if prefixLen > maxLen {
			maxLen = prefixLen
		}
		clients = append(clients, client)
	}
	for err := range errCh {
		return errors.Wrap(err, "connecting to clients failed")
	}

	// Run command or run multiple commands defined by target sequentially.
	for _, cmd := range commands {
		// Translate command into task(s).
		tasks, err := CreateTasks(cmd, clients, env, sup.Args)
		if err != nil {
			return errors.Wrap(err, "creating task failed")
		}

		// Run tasks sequentially.
		for _, task := range tasks {
			var writers []io.Writer
			var wg sync.WaitGroup

			//run shellcheck
			if e := svc.Lobby.Shellcheck.Check(task); e != nil {
				return oops.Trace("EBCC9B9D-9C3B-4C6D-A5F6-F54E151D2848").
					Hint("running shellcheck").
					Wrap(e)
			}

			// Run tasks on the provided clients.
			for _, c := range task.Clients {
				var prefix string
				var prefixLen int
				if sup.prefix {
					prefix, prefixLen = c.Prefix()
					if len(prefix) < maxLen { // Left padding.
						prefix = strings.Repeat(" ", maxLen-prefixLen) + prefix
					}
				}

				if len(cmd.Upload) > 0 {
					for _, upload_command := range cmd.Upload {
						if err := c.Upload(upload_command.Src, upload_command.Dst); err != nil {
							return errors.Wrap(err, prefix+"uploading files failed")
						}
					}
				}

				err := c.Run(task)
				if err != nil {
					return errors.Wrap(err, prefix+"task failed")
				}

				// Copy over tasks's STDOUT.
				wg.Add(1)
				go func(c entity.ClientFacade) {
					defer wg.Done()
					_, err := io.Copy(os.Stdout, prefixer.New(c.Stdout(), prefix))
					if err != nil && err != io.EOF {
						// TODO: io.Copy() should not return io.EOF at all.
						// Upstream bug? Or prefixer.WriteTo() bug?
						fmt.Fprintf(os.Stderr, "%v", errors.Wrap(err, prefix+"reading STDOUT failed"))
					}
				}(c)

				// Copy over tasks's STDERR.
				wg.Add(1)
				go func(c entity.ClientFacade) {
					defer wg.Done()
					_, err := io.Copy(os.Stderr, prefixer.New(c.Stderr(), prefix))
					if err != nil && err != io.EOF {
						fmt.Fprintf(os.Stderr, "%v", errors.Wrap(err, prefix+"reading STDERR failed"))
					}
				}(c)

				writers = append(writers, c.Stdin())
			}

			// Copy over task's STDIN.
			if task.Input != nil {
				go func() {
					writer := io.MultiWriter(writers...)
					_, err := io.Copy(writer, task.Input)
					if err != nil && err != io.EOF {
						fmt.Fprintf(os.Stderr, "%v", errors.Wrap(err, "copying STDIN failed"))
					}
					// TODO: Use MultiWriteCloser (not in Stdlib), so we can writer.Close() instead?
					for _, c := range clients {
						c.WriteClose()
					}
				}()
			}

			// Catch OS signals and pass them to all active clients.
			trap := make(chan os.Signal, 1)
			signal.Notify(trap, os.Interrupt)
			go func() {
				for {
					select {
					case sig, ok := <-trap:
						if !ok {
							return
						}
						for _, c := range task.Clients {
							err := c.Signal(sig)
							if err != nil {
								fmt.Fprintf(os.Stderr, "%v", errors.Wrap(err, "sending signal failed"))
							}
						}
					}
				}
			}()

			// Wait for all I/O operations first.
			wg.Wait()

			// Make sure each client finishes the task, return on failure.
			for _, c := range task.Clients {
				wg.Add(1)
				go func(c entity.ClientFacade) {
					defer wg.Done()
					if err := c.Wait(); err != nil {
						var prefix string
						if sup.prefix {
							var prefixLen int
							prefix, prefixLen = c.Prefix()
							if len(prefix) < maxLen { // Left padding.
								prefix = strings.Repeat(" ", maxLen-prefixLen) + prefix
							}
						}
						if e, ok := err.(*ssh.ExitError); ok && e.ExitStatus() != 15 {
							// TODO: Store all the errors, and print them after Wait().
							fmt.Fprintf(os.Stderr, "%s%v\n", prefix, e)
							os.Exit(e.ExitStatus())
						}
						fmt.Fprintf(os.Stderr, "%s%v\n", prefix, err)

						// TODO: Shouldn't os.Exit(1) here. Instead, collect the exit statuses for later.
						os.Exit(1)
					}
				}(c)
			}

			// Wait for all commands to finish.
			wg.Wait()

			// Stop catching signals for the currently active clients.
			signal.Stop(trap)
			close(trap)
		}
	}

	return nil
}

func connectToHosts(network *entity.Network, wg *sync.WaitGroup, env string, errCh chan error, clientCh chan entity.ClientFacade, bastion *gateway.SSHClient) {
	for i, host := range network.Hosts {
		wg.Add(1)
		go func(i int, host entity.NetworkHost) {
			defer wg.Done()

			// localhost client
			if host.Host == "localhost" || host.Host == "127.0.0.1" {
				local := &gateway.LocalhostClient{
					Env: env + `export SUP_HOST="` + host.Host + `";`,
				}
				if err := local.Connect(host); err != nil {
					errCh <- errors.Wrap(err, "connecting to localhost failed")
					return
				}
				clientCh <- local
				return
			}

			// SSH client
			pass := host.Password
			if pass == "" && network.Password != "" {
				pass = network.Password
			}

			remote := &gateway.SSHClient{
				Env:      env + `export SUP_HOST="` + host.Host + `";`,
				User:     network.User,
				Color:    entity.Colors[i%len(entity.Colors)],
				Password: pass,
			}

			if bastion != nil {
				if err := remote.ConnectWith(host, bastion.DialThrough); err != nil {
					errCh <- errors.Wrap(err, "connecting to remote host through bastion failed")
					return
				}
			} else {
				if err := remote.Connect(host); err != nil {
					errCh <- errors.Wrap(err, "connecting to remote host failed")
					return
				}
			}
			clientCh <- remote
		}(i, host)
	}
	return
}

func (*Stackup) connectToBationHost(network *entity.Network) (*gateway.SSHClient, error) {
	var bastion *gateway.SSHClient
	if network.Bastion != "" {
		bastion = &gateway.SSHClient{}
		bastionHost := entity.NetworkHost{
			Host: network.Bastion,
		}
		if err := bastion.Connect(bastionHost); err != nil {
			return nil, errors.Wrap(err, "connecting to bastion failed")
		}
	}
	return bastion, nil
}

func (sup *Stackup) Debug(value bool) {
	sup.debug = value
}

func (sup *Stackup) Prefix(value bool) {
	sup.prefix = value
}
