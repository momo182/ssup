package usecase

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"

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
	l := kemba.New("uc::Stackup::Run").Printf
	ctx, cancel := context.WithCancel(context.Background()) // Создать контекст
	defer cancel()                                          // На всякий случай, если не отменим явно

	if len(commands) == 0 {
		return sup.WrapError("1DF987CA-1E81-4C48-9163-DDAEA0C0CDF7", "no commands to be run", errors.New("no commands provided"))
	}

	bastion, err := sup.ConnectToBastionHostHelper(network)
	if err != nil {
		return sup.WrapError("60267BB8-0049-4819-BA06-30EFDFE161AD", "connecting to bastion host failed", err)
	}

	l("connected to bastion")

	clients, maxLen, failedHosts := sup.ConnectToHostsHelper(network, bastion)
	if len(failedHosts) != 0 {
		for name := range failedHosts {
			fmt.Printf("connecion failed: %s\n", name)
		}
		return sup.WrapError("B9E27F42-9351-4F36-9174-4B1F7B0B97D5", "connecting to clients failed", err)
	}

	// if err != nil {
	// 	return sup.WrapError("B9E27F42-9351-4F36-9174-4B1F7B0B97D5", "connecting to clients failed", err)
	// }

	l("connected to clients")
	l(dump.Format(clients))

	for _, cmd := range commands {
		l("processing command:\n%v", dump.Format(cmd))
		tasks, err := CreateTasks(cmd, clients, envVars, sup.Args)
		if err != nil {
			return sup.WrapError("75F613AD-0E35-4FA0-A9D5-5A44B0C4EB08", "creating tasks failed", err)
		}

		for _, task := range tasks {
			if err := sup.runTask(ctx, cancel, task, cmd, maxLen, isMakefileMode); err != nil {
				return err
			}
		}
	}

	l("closing all clients")
	for _, client := range clients {
		if err := client.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Helper function to wrap errors consistently.
func (sup *Stackup) WrapError(traceID, hint string, err error) error {
	return oops.Trace(traceID).Hint(hint).Wrap(err)
}

// Helper function to clean up temporary files.
func (sup *Stackup) cleanupTempFile(file *os.File) {
	l := kemba.New("uc::Stackup::cleanupTempFile").Printf
	l("set deferred cleanup")
	file.Close()
	try.E(fsutil.Remove(file.Name()))
}

// Helper function to connect to the bastion host.
func (sup *Stackup) ConnectToBastionHostHelper(network *entity.Network) (entity.ClientFacade, error) {
	l := kemba.New("uc::Stackup::connectToBastionHostHelper").Printf
	l("about to connect to bastion:\n%v", network.Hosts)
	return sup.connectToBastionHost(network) // Assuming this is the actual method on the Stackup struct.
}

// ConnectToHostsHelper is a helper function to connect to all hosts.
func (sup *Stackup) ConnectToHostsHelper(network *entity.Network, bastion entity.ClientFacade) ([]entity.ClientFacade, int, []string) {
	l := kemba.New("uc::Stackup::connectToHostsHelper").Printf

	l(dump.Format(network))
	l("about to connect to hosts:\n%v", network.Hosts)

	clients, hostnamesWithErr := connectToHosts(network, bastion.(*clientSSH.RemoteClient))

	if len(hostnamesWithErr) != 0 {
		for _, hostnameWithErr := range hostnamesWithErr {
			l("hostname failed to connect: %v", hostnameWithErr)
		}
		return nil, 0, hostnamesWithErr
	}

	if len(clients) == 0 {
		//bail
		return nil, 0, []string{"all"}
	}

	l("hots after connect fn:\n%s", dump.Format(clients))
	maxLen := 0
	for _, client := range clients {
		_, prefixLen := client.Prefix()
		if prefixLen > maxLen {
			maxLen = prefixLen
		}
	}

	l("connected to all clients")
	return clients, maxLen, nil
}

// Helper function to run a task.
func (sup *Stackup) runTask(ctx context.Context, cancel context.CancelFunc, task *entity.Task, cmd *entity.Command, maxLen int, isMakefileMode bool) error {
	l := kemba.New("uc::Stackup::runTask").Printf
	l("task:\n%v", task)
	l("clients:\n%v", task.Clients)

	totalClients := len(task.Clients)
	if totalClients == 0 {
		return fmt.Errorf("no clients to run task on")
	}

	// Канал для сбора первой ошибки
	errChan := make(chan error, totalClients)
	var wg sync.WaitGroup

	// Флаг для отслеживания, была ли ошибка и отмена уже инициирована
	var canceled int32 // 0 = false, 1 = true

	setCanceled := func() {
		if atomic.CompareAndSwapInt32(&canceled, 0, 1) {
			l("Canceling context due to error")
			cancel() // Отменяем контекст при первой ошибке
		}
	}

	for _, c := range task.Clients {
		l("client: %s", c.GetHost())
		prefix := sup.getPrefix(c, maxLen, isMakefileMode)

		// Проверка контекста перед началом работы с клиентом
		select {
		case <-ctx.Done():
			// Контекст уже отменен, пропускаем клиента
			l("Context canceled before starting work on client %s", c.GetHost())
			continue // Или return ctx.Err() если нужно остановить весь runTask сразу
		default:
		}

		// Handle uploads if any.
		l("handling uploads for %s", c.GetHost())
		if err := sup.handleUploads(c, cmd.Upload); err != nil {
			setCanceled() // Отменяем при ошибке загрузки
			errChan <- errors.Wrap(err, prefix+"upload failed")
			// Не возвращаем сразу, даем возможность запустить другие клиенты или дождаться уже запущенных
			continue // Или break если нужно остановить обработку других клиентов в этом runTask
		}

		// Run the task on the client.
		l("running task on client")
		if err := c.Run(task); err != nil {
			return errors.Wrap(err, prefix+"task failed")
		}

		// Handle I/O for the client.
		l("handling I/O")
		sup.handleIO(c, prefix, &wg)
	}

	// Handle task input by writing to the clients' Stdin.
	if task.Input != nil {
		var writers []io.Writer
		for _, c := range task.Clients {
			writers = append(writers, c.Stdin())
		}

		sup.handleTaskInput(task, writers, task.Clients)
	}

	// Wait for all I/O operations to complete.
	wg.Wait()

	// Wait for all clients to finish the task.
	if err := sup.waitForClients(task.Clients, maxLen, isMakefileMode); err != nil {
		return errors.Wrap(err, "task failed")
	}

	for _, c := range task.Clients {
		l("client downloads check: %s", c.GetHost())
		prefix := sup.getPrefix(c, maxLen, isMakefileMode)

		// Проверка контекста перед началом работы с клиентом
		select {
		case <-ctx.Done():
			// Контекст уже отменен, пропускаем клиента
			l("Context canceled before starting work on client %s", c.GetHost())
			continue // Или return ctx.Err() если нужно остановить весь runTask сразу
		default:
		}

		// Handle uploads if any.
		if totalClients == 1 {
			for counter, command := range cmd.Fetch {
				command.Flat = true
				cmd.Fetch[counter] = command
			}
		}

		l("handling fetch for %s", c.GetHost())
		if err := sup.handleDownloads(c, cmd.Fetch); err != nil {
			setCanceled() // Отменяем при ошибке загрузки
			errChan <- errors.Wrap(err, prefix+"fetch failed")
			// Не возвращаем сразу, даем возможность запустить другие клиенты или дождаться уже запущенных
			continue // Или break если нужно остановить обработку других клиентов в этом runTask
		}
	}

	return nil
}

// Helper function to handle uploads.
func (sup *Stackup) handleUploads(client entity.ClientFacade, uploads []*entity.UploadOrder) error {
	l := kemba.New("uc::Stackup::handleUploads").Printf
	for _, upload := range uploads {
		l("upload command: %v", upload.Src)
		if err := client.Upload(upload.Src, upload.Dst); err != nil {
			return sup.WrapError("28D38F66-3258-4578-A275-7D70F3765C0A", "running upload command failed", err)
		}
	}
	return nil
}

// Helper function to handle downloads.
func (sup *Stackup) handleDownloads(client entity.ClientFacade, downloads []*entity.DownloadOrder) error {
	l := kemba.New("uc::Stackup::handleDownloads").Printf
	for _, download := range downloads {
		l("download command: %v", download.Src)
		if err := client.Download(download.Src, download.Dst, false, download.Flat); err != nil {
			return sup.WrapError("606ED9CC-9930-42FD-9189-927278EFFD7A", "running download command failed", err)
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

func connectToHosts(network *entity.Network, bastion *clientSSH.RemoteClient) ([]entity.ClientFacade, []string) {
	l := kemba.New("usecase::run::connectToHosts").Printf
	l("will range over hosts: %v", len(network.Hosts))

	type result struct {
		client entity.ClientFacade
		err    error
		host   string
	}

	results := make(chan result, len(network.Hosts))

	// Запускаем горутины для подключения
	for i := range network.Hosts {
		go func(host entity.NetworkHost) {
			var client entity.ClientFacade
			var err error

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

				err = local.Connect(host)
				client = local
			} else {
				l("found remote host: %s", host.Host)

				pass := host.Password
				if pass == "" && network.Password != "" {
					pass = network.Password
				}

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

				if bastion != nil {
					l("bastion is set, trying it now")
					err = remote.ConnectWith(host, bastion.DialThrough)
				} else {
					l("connecting via direct connection")
					err = remote.Connect(host)
				}
				client = remote
			}

			results <- result{client: client, err: err, host: host.Host}
		}(network.Hosts[i])
	}

	// Собираем результаты
	clients := []entity.ClientFacade{}
	errors := []string{}

	for i := 0; i < len(network.Hosts); i++ {
		res := <-results
		if res.err != nil {
			l("failed to connect to %s: %v", res.host, res.err)
			errors = append(errors, res.host)
		} else {
			clients = append(clients, res.client)
		}
	}

	l("connected to all hosts:\n%s", dump.Format(clients))
	return clients, errors
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
