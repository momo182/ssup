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
)

type Stackup struct {
	conf   *entity.Supfile
	debug  bool
	prefix bool
	Args   *entity.InitialArgs
}

// NewStackup creates a new Stackup instance.
func NewStackup(conf *entity.Supfile) *Stackup {
	return &Stackup{
		conf: conf,
	}
}

// Run runs set of commands on multiple hosts defined by network sequentially.
//
//	TODO: This megamoth method needs a big refactor and should be split
//	to multiple smaller methods.
func (sup *Stackup) Run(isMakefileMode bool, network *entity.Network, envVars entity.EnvList, commands ...*entity.Command) error {
	logger := kemba.New("uc::Stackup::Run").Printf

	if len(commands) == 0 {
		return sup.wrapError("1DF987CA-1E81-4C48-9163-DDAEA0C0CDF7", "no commands to be run", errors.New("no commands provided"))
	}

	bastion, err := sup.connectToBastionHostHelper(network, logger)
	if err != nil {
		return sup.wrapError("60267BB8-0049-4819-BA06-30EFDFE161AD", "connecting to bastion host failed", err)
	}
	logger("connected to bastion")

	clients, maxLen, err := sup.connectToHostsHelper(network, bastion, logger)
	if err != nil {
		return sup.wrapError("B9E27F42-9351-4F36-9174-4B1F7B0B97D5", "connecting to clients failed", err)
	}
	logger("connected to clients")

	for _, cmd := range commands {
		logger("processing command:\n%v", dump.Format(cmd))
		tasks, err := CreateTasks(cmd, clients, envVars, sup.Args)
		if err != nil {
			return sup.wrapError("75F613AD-0E35-4FA0-A9D5-5A44B0C4EB08", "creating tasks failed", err)
		}

		for _, task := range tasks {
			if err := sup.runTask(task, cmd, clients, maxLen, isMakefileMode, logger); err != nil {
				return err
			}
		}
	}

	return nil
}

// Helper function to wrap errors consistently.
func (sup *Stackup) wrapError(traceID, hint string, err error) error {
	return oops.Trace(traceID).Hint(hint).Wrap(err)
}

// Helper function to clean up temporary files.
func (sup *Stackup) cleanupTempFile(file *os.File, logger func(string, ...interface{})) {
	logger("set deferred cleanup")
	file.Close()
	try.E(fsutil.Remove(file.Name()))
}

// Helper function to connect to the bastion host.
func (sup *Stackup) connectToBastionHostHelper(network *entity.Network, logger func(string, ...interface{})) (entity.ClientFacade, error) {
	logger("about to connect to bastion:\n%v", network.Hosts)
	return sup.connectToBastionHost(network) // Assuming this is the actual method on the Stackup struct.
}

// Helper function to connect to all hosts.
func (sup *Stackup) connectToHostsHelper(network *entity.Network, bastion entity.ClientFacade, logger func(string, ...interface{})) ([]entity.ClientFacade, int, error) {
	var connectWg sync.WaitGroup
	clientCh := make(chan entity.ClientFacade, len(network.Hosts))
	errCh := make(chan error, len(network.Hosts))

	logger("about to connect to hosts:\n%v", network.Hosts)
	connectToHosts(network, &connectWg, errCh, clientCh, bastion.(*clientSSH.RemoteClient))
	connectWg.Wait()
	close(clientCh)
	close(errCh)

	var clients []entity.ClientFacade
	maxLen := 0
	for client := range clientCh {
		if remote, ok := client.(*clientSSH.RemoteClient); ok {
			defer remote.Close()
		}
		_, prefixLen := client.Prefix()
		if prefixLen > maxLen {
			maxLen = prefixLen
		}
		clients = append(clients, client)
	}

	for err := range errCh {
		return nil, 0, err
	}

	logger("connected to all clients")
	return clients, maxLen, nil
}

// Helper function to run a task.
func (sup *Stackup) runTask(task *entity.Task, cmd *entity.Command, clients []entity.ClientFacade, maxLen int, isMakefileMode bool, logger func(string, ...interface{})) error {
	logger("task: %s", dump.Format(task))
	var wg sync.WaitGroup

	for _, c := range task.Clients {
		logger("client: %s", c.GetHost())
		prefix := sup.getPrefix(c, maxLen, isMakefileMode)

		// Handle uploads if any.
		logger("handling uploads")
		if err := sup.handleUploads(c, cmd.Upload, logger); err != nil {
			return err
		}

		// Run the task on the client.
		logger("running task on client")
		if err := c.Run(task); err != nil {
			return errors.Wrap(err, prefix+"task failed")
		}

		// Handle I/O for the client.
		logger("handling I/O")
		sup.handleIO(c, prefix, &wg)
	}

	// Handle task input by writing to the clients' Stdin.
	if task.Input != nil {
		var writers []io.Writer
		for _, c := range task.Clients {
			writers = append(writers, c.Stdin())
		}

		sup.handleTaskInput(task, writers, clients)
	}

	// Wait for all I/O operations to complete.
	wg.Wait()

	// Wait for all clients to finish the task.
	return sup.waitForClients(task.Clients, maxLen, isMakefileMode)
}

// Helper function to handle uploads.
func (sup *Stackup) handleUploads(client entity.ClientFacade, uploads []*entity.Upload, logger func(string, ...interface{})) error {
	for _, upload := range uploads {
		logger("upload command: %v", upload.Src)
		if err := client.Upload(upload.Src, upload.Dst); err != nil {
			return sup.wrapError("28D38F66-3258-4578-A275-7D70F3765C0A", "running upload command failed", err)
		}
	}
	return nil
}

// Helper function to handle I/O operations.
func (sup *Stackup) handleIO(client entity.ClientFacade, prefix string, wg *sync.WaitGroup) {
	l := kemba.New("uc::run::handleIO").Printf
	l("handling I/O")
	l("negative checks")

	wg.Add(2)
	l("adding two wgs")
	go func() {
		defer wg.Done()
		l("wg for stdout")
		sup.copyOutput(client.Stdout(), os.Stdout, prefix, "reading STDOUT failed")
	}()
	go func() {
		defer wg.Done()
		l("wg for stderr")
		sup.copyOutput(client.Stderr(), os.Stderr, prefix, "reading STDERR failed")
	}()
}

// Helper function to copy output with prefix.
func (sup *Stackup) copyOutput(src io.Reader, dst io.Writer, prefix, errMsg string) {
	_, err := io.Copy(dst, prefixer.New(src, prefix))
	if err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "%v", errors.Wrap(err, prefix+errMsg))
	}
}

// Helper function to handle task input.
func (sup *Stackup) handleTaskInput(task *entity.Task, writers []io.Writer, clients []entity.ClientFacade) {
	if len(clients) == 1 {
		if clients[0].GetHost() == "localhost" {
			return
		}
	}
	if task.Input != nil {
		go func() {
			writer := io.MultiWriter(writers...)
			_, err := io.Copy(writer, task.Input)
			if err != nil && err != io.EOF {
				fmt.Fprintf(os.Stderr, "%v", errors.Wrap(err, "copying STDIN failed"))
			}
			for _, c := range clients {
				c.WriteClose()
			}
		}()
	}
}

// Helper function to handle signals.
func (sup *Stackup) handleSignals(task *entity.Task, logger func(string, ...interface{})) {
	trap := make(chan os.Signal, 1)
	signal.Notify(trap, os.Interrupt)
	go func() {
		for sig := range trap {
			logger("sending signal %s to %s\n", sig.String(), dump.Format(task))
			for _, c := range task.Clients {
				if err := c.Signal(sig); err != nil {
					fmt.Fprintf(os.Stderr, "%v", errors.Wrap(err, "sending signal failed"))
				}
			}
		}
	}()
	defer signal.Stop(trap)
	defer close(trap)
}

// Helper function to wait for clients to finish.
func (sup *Stackup) waitForClients(clients []entity.ClientFacade, maxLen int, isMakefileMode bool) error {
	var wg sync.WaitGroup
	for _, c := range clients {
		wg.Add(1)
		go func(c entity.ClientFacade) {
			defer wg.Done()
			if err := c.Wait(); err != nil {
				prefix := sup.getPrefix(c, maxLen, isMakefileMode)
				fmt.Fprintf(os.Stderr, "%s%v\n", prefix, err)
				os.Exit(1)
			}
		}(c)
	}
	wg.Wait()
	return nil
}

// Helper function to get the prefix for a client.
func (sup *Stackup) getPrefix(client entity.ClientFacade, maxLen int, isMakefileMode bool) string {
	if !sup.prefix || isMakefileMode {
		return ""
	}
	prefix, prefixLen := client.Prefix()
	if len(prefix) < maxLen {
		prefix = strings.Repeat(" ", maxLen-prefixLen) + prefix
	}
	return prefix
}

func connectToHosts(network *entity.Network, wg *sync.WaitGroup, errCh chan error, clientCh chan entity.ClientFacade, bastion *clientSSH.RemoteClient) {
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
			remote := &clientSSH.RemoteClient{
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

func (*Stackup) connectToBastionHost(network *entity.Network) (*clientSSH.RemoteClient, error) {
	l := kemba.New("usecase::run::connectToBastionHost").Printf

	l("prepping ssh client to bastion: %s", network.Bastion)
	var bastion *clientSSH.RemoteClient
	if network.Bastion != "" {
		bastion = &clientSSH.RemoteClient{}
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
