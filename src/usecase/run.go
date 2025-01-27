package usecase

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/clok/kemba"
	"github.com/dsnet/try"
	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/fsutil"
	"github.com/goware/prefixer"
	"github.com/momo182/ssup/src/entity"
	clientLocal "github.com/momo182/ssup/src/gateway/localhost"
	clientSSH "github.com/momo182/ssup/src/gateway/ssh"
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

// NewStackup creates a new Stackup instance.
func NewStackup(conf *entity.Supfile) (*Stackup, error) {
	return &Stackup{
		conf: conf,
	}, nil
}

// Run runs set of commands on multiple hosts defined by network sequentially.
//
//	TODO: This megamoth method needs a big refactor and should be split
//	to multiple smaller methods.
func (sup *Stackup) Run(network *entity.Network, envVars entity.EnvList, commands ...*entity.Command) error {
	l := kemba.New("usecase::Stackup.Run").Printf
	rcloneTmpCfg := try.E1(fsutil.TempFile("", "rclone_configuration.*.cfg"))
	rcloneTmpCfg.WriteString("")

	defer func() {
		l("set deferred cleanup")
		rcloneTmpCfg.Close()
		try.E(fsutil.Remove(rcloneTmpCfg.Name()))
	}()

	if len(commands) == 0 {
		return oops.Trace("1DF987CA-1E81-4C48-9163-DDAEA0C0CDF7").
			Hint("check how many commands").
			Wrap(errors.New("no commands to be run"))
	}

	env := envVars

	// Create clients for every host (either SSH or Localhost).
	l("about to connect to bastion:\n%v", network.Hosts)
	bastion, e := sup.connectToBastionHost(network)
	if e != nil {
		return oops.
			Trace("60267BB8-0049-4819-BA06-30EFDFE161AD").
			Hint("connecting to bastion host").
			Wrap(e)
	}

	var connectWg sync.WaitGroup
	clientCh := make(chan entity.ClientFacade, len(network.Hosts))
	errCh := make(chan error, len(network.Hosts))

	l("about to connect to hosts:\n%v", network.Hosts)
	connectToHosts(network, &connectWg, errCh, clientCh, bastion)
	connectWg.Wait()
	close(clientCh)
	close(errCh)

	maxLen := 0
	var clients []entity.ClientFacade = make([]entity.ClientFacade, 0)
	for client := range clientCh {
		if remote, ok := client.(*clientSSH.SSHClient); ok {
			defer remote.Close()
		}
		_, prefixLen := client.Prefix()
		if prefixLen > maxLen {
			maxLen = prefixLen
		}
		clients = append(clients, client)
	}
	for err := range errCh {
		return oops.Trace("B9E27F42-9351-4F36-9174-4B1F7B0B97D5").
			Hint("connecting to clients failed").
			Wrap(err)
	}
	l("connected to all clients")

	// Run command or run multiple commands defined by target sequentially.
	for _, cmd := range commands {
		// Translate command into task(s).
		l("command: %v", cmd.Name)
		l("will create tasks")
		tasks, err := CreateTasks(cmd, clients, env, sup.Args)
		if err != nil {
			return oops.Trace("75F613AD-0E35-4FA0-A9D5-5A44B0C4EB08").
				Hint("creating task failed").
				With("tasks", tasks).
				Wrap(e)
		}

		// Run tasks sequentially.
		for _, task := range tasks {
			l("task: %v", task)
			var writers []io.Writer
			var wg sync.WaitGroup

			// Run tasks on the provided clients.
			for _, c := range task.Clients {
				var prefix string
				var prefixLen int
				c.SetRcloneCfg(rcloneTmpCfg.Name())

				if sup.prefix {
					prefix, prefixLen = c.Prefix()
					if len(prefix) < maxLen { // Left padding.
						prefix = strings.Repeat(" ", maxLen-prefixLen) + prefix
					}
				}

				if len(cmd.Upload) > 0 {
					for _, uploadCommand := range cmd.Upload {
						if err := c.Upload(uploadCommand.Src, uploadCommand.Dst, rcloneTmpCfg.Name()); err != nil {
							return oops.Trace("28D38F66-3258-4578-A275-7D70F3765C0A").
								Hint("running upload command").
								With("upload_command", uploadCommand).
								Wrap(e)
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

						l("sending signal %s to %s\n", sig.String(), dump.Format(task))

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

func connectToHosts(network *entity.Network, wg *sync.WaitGroup, errCh chan error, clientCh chan entity.ClientFacade, bastion *clientSSH.SSHClient) {
	l := kemba.New("usecase::run::connectToHosts").Printf
	l("will range over hosts: %v", len(network.Hosts))
	for i, host := range network.Hosts {
		wg.Add(1)
		go func(i int, host entity.NetworkHost) {
			defer wg.Done()

			// localhost client
			if host.Host == "localhost" || host.Host == "127.0.0.1" {
				l("found localhost")
				envStore := new(entity.EnvList)
				envStore.Set("SUP_HOST", host.Host)
				local := &clientLocal.LocalhostClient{
					Env: envStore,
				}

				if host.Tube != "" {
					l("will inject tube: %s", host.Tube)
					local.SetTube(host.Tube)
				}

				l("about to connect to localhost")
				if err := local.Connect(host); err != nil {
					errCh <- errors.Wrap(err, "connecting to localhost failed")
					return
				}
				clientCh <- local
				return
			}
			l("found remote host: %s", host.Host)

			// SSH client
			l("password check for host")
			pass := host.Password
			if pass == "" && network.Password != "" {
				pass = network.Password
			}

			l("filling in user,env and creds")
			envStore := new(entity.EnvList)
			envStore.Set("SUP_HOST", host.Host)
			remote := &clientSSH.SSHClient{
				Env:      envStore,
				User:     network.User,
				Color:    entity.Colors[i%len(entity.Colors)],
				Password: pass,
			}

			if host.Tube != "" {
				l("will inject tube: %s", host.Tube)
				remote.SetTube(host.Tube)
			}

			l("about to connect to remote host")

			if bastion != nil {
				l("bastion is set, trying it now")
				if err := remote.ConnectWith(host, bastion.DialThrough); err != nil {
					errCh <- errors.Wrap(err, "connecting to remote host through bastion failed")
					return
				}
			} else {
				l("connecting via direct connection")
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

func (*Stackup) connectToBastionHost(network *entity.Network) (*clientSSH.SSHClient, error) {
	l := kemba.New("usecase::run::connectToBastionHost").Printf

	l("prepping ssh client to bastion: %s", network.Bastion)
	var bastion *clientSSH.SSHClient
	if network.Bastion != "" {
		bastion = &clientSSH.SSHClient{}
		bastionHost := entity.NetworkHost{
			Host: network.Bastion,
		}
		l("launch client connection to bastion")
		if err := bastion.Connect(bastionHost); err != nil {
			return nil, errors.Wrap(err, "connecting to bastion failed")
		}
	}
	l("done with bastion connection")
	return bastion, nil
}

// Debug sets whether or not to print debug messages
func (sup *Stackup) Debug(value bool) {
	sup.debug = value
}

// Prefix sets the host prefix for printing output from it
func (sup *Stackup) Prefix(value bool) {
	sup.prefix = value
}
