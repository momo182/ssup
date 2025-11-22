package ssh

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/clok/kemba"
	"github.com/goforj/godump"
	"github.com/gookit/goutil/fsutil"
	"github.com/momo182/ssup/src/entity"
	keys "github.com/momo182/ssup/src/gateway/keyauthprovider"
	ns "github.com/momo182/ssup/src/gateway/namespace"
	sudo "github.com/momo182/ssup/src/gateway/sudo_formatter"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// RemoteClient is a wrapper over the SSH connection/sessions.
type RemoteClient struct {
	conn         *ssh.Client
	sess         *ssh.Session
	ConnectOrder entity.ConnectOrder
	User         string
	Host         string
	Password     string
	remoteStdin  io.WriteCloser
	remoteStdout io.Reader
	remoteStderr io.Reader
	connOpened   bool
	sessOpened   bool
	running      bool
	Env          *entity.EnvList
	Color        string
	nameSpace    string
	Inventory    *entity.Inventory
	// encryptedPassword []byte
	// encryption        string
}

// init initializes the package.
func init() {
	// Initialization logic can be added here if needed.
}

// ErrConnect describes a connection error.
type ErrConnect struct {
	User   string
	Host   string
	Reason string
}

// Error returns a formatted string representation of the connection error.
func (e ErrConnect) Error() string {
	return fmt.Sprintf(`Connect("%v@%v"): %v`, e.User, e.Host, e.Reason)
}

// ErrInv describes an inventory error.
type ErrInv struct {
	User   string
	Host   string
	Reason string
}

// Error returns a formatted string representation of the inventory error.
func (e ErrInv) Error() string {
	return fmt.Sprintf(`Connect("%v@%v"): %v`, e.User, e.Host, e.Reason)
}

func (client *RemoteClient) String() string {
	return fmt.Sprintf("RemoteClient{Host: %v, Tube: %v}", client.Host, client.nameSpace)
}

// GetHost returns the host of the SSHClient.
func (client *RemoteClient) GetHost() string {
	return client.Host
}

// GetTube returns the tube of the SSHClient.
func (client RemoteClient) GetTube() string {
	return client.nameSpace
}

// GetConnection returns the client configuration of the SSHClient.
func (client RemoteClient) GetSSHConfig() *ssh.ClientConfig {
	return client.ConnectOrder.ClientConfig
}

// GetConnection returns the client configuration of the SSHClient.
func (client RemoteClient) GetConnection() *ssh.Client {
	return client.conn
}

// SetConnection sets the SSH client connection of the SSHClient.
func (c *RemoteClient) SetConnection(client *ssh.Client) {
	c.conn = client
	c.connOpened = true
}

// SetTube sets the tube of the SSHClient.
func (client *RemoteClient) SetTube(name string) {
	client.nameSpace = name
}

// GetPassword returns the password of the SSHClient.
func (client *RemoteClient) GetPassword() string {
	return client.Password
}

// SetPassword sets the password of the SSHClient.
func (client *RemoteClient) SetPassword(pwd string) {
	client.Password = pwd
}

// parseHost parses and normalizes <user>@<host:port> from a given string.
func (client *RemoteClient) parseHost(host string) error {
	client.Host = host

	// Remove extra "ssh://" schema
	if len(client.Host) > 6 && client.Host[:6] == "ssh://" {
		client.Host = client.Host[6:]
	}

	// Split by the last "@", since there may be an "@" in the username.
	if at := strings.LastIndex(client.Host, "@"); at != -1 {
		client.User = client.Host[:at]
		client.Host = client.Host[at+1:]
	}

	// Add default user, if not set
	if client.User == "" {
		u, err := user.Current()
		if err != nil {
			return err
		}
		client.User = u.Username
	}

	if strings.Index(client.Host, "/") != -1 {
		return ErrConnect{client.User, client.Host, "unexpected slash in the host URL"}
	}

	// Add default port, if not set
	if strings.Index(client.Host, ":") == -1 {
		client.Host += ":22"
	}

	return nil
}

// SSHDialFunc can dial an ssh server and return a client
type SSHDialFunc func(net, addr string, config *ssh.ClientConfig) (*ssh.Client, error)

// Connect creates SSH connection to a specified host.
// It expects the host of the form "[ssh://]host[:port]".
func (client *RemoteClient) Connect(host entity.NetworkHost) error {
	l := kemba.New("gw::ssh::SSHClient.Connect").Printf

	err := client.parseHost(host.Host)
	if err != nil {
		return err
	}

	l("checking password auth")
	authMethods := keys.SetupAuthMethods(host)

	l("creating config")
	// l(godump.DumpStr(host, authMethods))
	config := ssh.ClientConfig{
		User:            client.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	// l(godump.DumpStr(config))

	l("creating connect order")
	connectOrder := entity.ConnectOrder{
		Host:         host.Host,
		ClientConfig: &config,
	}
	// l(godump.DumpStr(connectOrder))

	client.ConnectOrder = connectOrder

	return client.ConnectWith(host, ssh.Dial)
}

func (client *RemoteClient) CopyDistFolders() error {
	l := kemba.New("gw::ssh::SSHClient.copyDistFolders").Printf
	l("copying dist folder")

	if client == nil {
		l("client is nil")
		return errors.New("client is nil")
	}

	if client.Inventory == nil {
		l("client inventory is nil")
		return errors.New("client inventory is nil")
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return errors.New("failed to get current directory")
	}

	srcPath := filepath.Join(currentDir, "dist")

	l("inspecting ./dist folder")
	if fsutil.PathExists(srcPath) {
		files, err := ListFiles(srcPath)
		if err != nil {
			return errors.New("failed to list files in dist folder")
		}

		if len(files) == 0 {
			return nil
		}
	}

	l(godump.DumpStr(client.Inventory))

	// check if dist of ssupdist is present and if so copy it
	// to remote host ~/.local/ssup/dist

	_, err = os.Stat(srcPath)
	if err != nil {
		l("dist folder not found at %v", srcPath)
		return nil
	}
	l("dist folder found at %v", srcPath)

	sep := "/"
	if client.Inventory.OsType != "Darwin" && client.Inventory.OsType != "Linux" {
		sep = "\\"
	}

	l("sep is: '%v'", sep)
	destPath := client.Inventory.Home + sep + ".local" + sep + "ssup" + sep + "dist"
	l("destPath is: '%v'")

	return client.Upload(srcPath, destPath)
}

// ConnectWith creates a SSH connection to a specified host. It will use dialer to establish the
// connection.
// TODO: Split Signers to its own method.
func (client *RemoteClient) ConnectWith(host entity.NetworkHost, dialer SSHDialFunc) error {
	l := kemba.New("gw::ssh::SSHClient.ConnectWith").Printf
	l("connecting to %v", host)
	var err error

	if client.connOpened {
		return fmt.Errorf("already connected")
	}
	config := client.ConnectOrder.ClientConfig
	if config == nil {
		return fmt.Errorf("ssh ClientConfig is nil")
	}

	config.Timeout = 500 * time.Millisecond

	l("creating ssh client")
	client.conn, err = dialer("tcp", client.Host, config)
	if err != nil {
		return ErrConnect{client.User, client.Host, err.Error()}
	}

	// TODO add inventory here
	inventory, err := GatherInventory(client.conn)
	if err != nil {
		return ErrInv{client.User, client.Host, err.Error()}
	}
	client.Inventory = inventory

	// // do dist copy here
	// err = copyDistFolders(c)
	// if err != nil {
	// 	return fmt.Errorf("failed to copy dist folders for remote client = %s", c.Host)
	// }

	client.connOpened = true
	l("done creating ssh client")

	// add namespace for the host
	l("adding namespace for host: %v", client.Host)
	ns.Namespaces.Add(client.Host)
	l("done adding namespace for host: %v", client.Host)
	return nil
}

func isConnected(client *ssh.Client) bool {
	_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
	return err == nil
}

// Run runs the task.Run command remotely on c.host.
func (c *RemoteClient) Run(task *entity.Task) error {
	l := kemba.New("gateway::ssh::Run").Printf
	l("negative programming checks")

	err := c.negativeChecksBeforeRun(task)
	if err != nil {
		return err
	}

	// test if connection is still alive
	if !isConnected(c.conn) {
		l("connection was dropped, probably by scp, will reconnect")
		config := c.GetSSHConfig()
		client, err := ssh.Dial("tcp", c.Host, config)
		if err != nil {
			return err
		}
		c.SetConnection(client)
	}

	l("append envs to client")
	if len(task.Env.Keys()) != 0 {
		for _, key := range task.Env.Keys() {
			c.Env.Set(key, task.Env.Get(key))
		}
	}

	l("setting pipes")
	sess, err := c.conn.NewSession()
	if err != nil {
		return err
	}

	c.remoteStdin, err = sess.StdinPipe()
	if err != nil {
		return err
	}

	c.remoteStdout, err = sess.StdoutPipe()
	if err != nil {
		return err
	}

	c.remoteStderr, err = sess.StderrPipe()
	if err != nil {
		return err
	}

	l("all pipes are set...")

	l("prepping tty")
	if task.TTY {
		// Set up terminal modes
		modes := ssh.TerminalModes{
			ssh.ECHO:          0,     // disable echoing
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}
		// get current terminal size
		width, height, err := terminal.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			return entity.ErrTask{Task: task, Reason: fmt.Sprintf("failed to get terminal size: %s", err)}
		}

		// Request pseudo terminal
		if err := sess.RequestPty("xterm", width, height, modes); err != nil {
			return entity.ErrTask{Task: task, Reason: fmt.Sprintf("request for pseudo terminal failed: %s", err)}
		}
		l("tty requested")
		// print user a message that we're requesing a TTY
		// and that the system awaits input from user
		fmt.Printf("(%s)>>MODE INTERACTIVE SET:\n", c.Host)
	}

	// actuall create command from task
	command := c.buildRemoteCommand(*task)
	l("built following command:\n%s", godump.DumpStr(command))

	if err := sess.Start(command); err != nil {
		return entity.ErrTask{Task: task, Reason: err.Error()}
	}

	c.sess = sess
	c.sessOpened = true
	c.running = true
	l("done with Run func")
	return nil
}

func (client *RemoteClient) negativeChecksBeforeRun(task *entity.Task) error {
	if task == nil {
		return errors.New("got nil task")
	}

	if client.conn == nil {
		return errors.New("ssh client not connected")
	}

	if client.sess != nil {
		return errors.New("ssh session somehow... connected")
	}

	if client.running {
		return fmt.Errorf("session already running")
	}

	if client.sessOpened {
		return fmt.Errorf("session already connected")
	}
	return nil
}

func (client *RemoteClient) GetShell() string {
	if client.Inventory.Bash {
		return "bash"
	}

	if client.Inventory.Sh {
		return "sh"
	}

	return ""
}

func (client *RemoteClient) GetInventory() *entity.Inventory {
	return client.Inventory
}

// Wait waits until the remote command finishes and exits.
// It closes the SSH session.
func (client *RemoteClient) Wait() error {
	l := kemba.New("gateway::ssh::SSHClient.Wait").Printf
	l("will wait for client: %s", client.Host)
	if client == nil {
		log.Panic("7782C3CF-5E8E-4740-9F7E-D68A9B2ED71C: no ssh client passed")
	}

	if !client.running {
		return fmt.Errorf("trying to wait on stopped session")
	}

	e := client.sess.Wait()
	if e != nil {
		return entity.ErrTask{Task: nil, Reason: e.Error()}
	}
	client.sess.Close()
	client.running = false
	client.sessOpened = false

	return client.FetchEnvsWithTar()
}

func (c *RemoteClient) FetchEnvsWithTar() error {
	l := kemba.New("gw::ssh::SSHClient.FetchEnvsWithTar").Printf
	l("will get remote envs")

	connectionUser := c.GetSSHConfig().User
	connUserHomeDir := "/home/" + connectionUser
	if connectionUser == "root" {
		connUserHomeDir = "/root"
	}

	remoteFilePath := filepath.Join(connUserHomeDir, entity.SSUP_WORK_FOLDER, "_tube_data")
	l("remote file path: %s", remoteFilePath)

	client, err := connectSFTP(c.GetConnection())
	if err != nil {
		return err
	}
	// defer client.Close()

	_, err = client.Stat(remoteFilePath)
	if err != nil {
		l("remote env file not found, skipping transfer")
		return nil
	}

	defer func() error {
		l("wiping tube data")
		err := client.Remove(remoteFilePath)
		if err != nil {
			return fmt.Errorf("failed to remove _tube_data on remote: %s", err.Error())
		}
		l("done wiping tube data")
		return nil
	}()

	// Open remote file
	localPath, err := fsutil.TempFile("", "SSUP_TEMP_*.tmp")
	defer fsutil.DeleteIfFileExist(localPath.Name())
	c.Download(remoteFilePath, localPath.Name(), false, true)
	remoteVars := fsutil.ReadAll(localPath)

	l("dump: 21D2239C-7DCD-4044-B30C-FFB5254EB0C2")
	l("%s", godump.DumpStr(string(remoteVars)))

	ns.Namespaces.SetFromEnvString(string(remoteVars), c.Host)
	debugData := ns.Namespaces.Get(c.Host)
	l("data:\n%s", godump.DumpStr(debugData))
	return nil
}

// DialThrough will create a new connection from the ssh server sc is connected to. DialThrough is an SSHDialer.
func (client *RemoteClient) DialThrough(net, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	conn, err := client.conn.Dial(net, addr)
	if err != nil {
		return nil, err
	}
	cl, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(cl, chans, reqs), nil
}

// Close closes the underlying SSH connection and session.
func (client *RemoteClient) Close() error {
	if client.sessOpened {
		client.sess.Close()
		client.sessOpened = false
	}
	if !client.connOpened {
		return fmt.Errorf("trying to close the already closed connection")
	}

	err := client.conn.Close()
	client.connOpened = false
	client.running = false

	return err
}

// Stdin sets remote stdin
func (client *RemoteClient) Stdin() io.WriteCloser {
	return client.remoteStdin
}

// Stderr sets remote stderr
func (client *RemoteClient) Stderr() io.Reader {
	return client.remoteStderr
}

// Stdout sets remote stdout
func (client *RemoteClient) Stdout() io.Reader {
	return client.remoteStdout
}

// Prefix sets prefix for printing
func (client *RemoteClient) Prefix() (string, int) {
	l := kemba.New("gateway::ssh::SSHClient.Prefix").Printf
	hostName := client.Host
	if strings.Contains(client.Host, ":") {
		hostName = client.Host[:strings.Index(client.Host, ":")]
	}

	host := client.User + "@" + hostName
	// host := "@" + hostName
	l("host: %s", host)
	return client.Color + host + entity.ResetColor + " " + "|" + " ", len(host) + 3
}

func (client *RemoteClient) Write(p []byte) (n int, err error) {
	if client.remoteStdin == nil {
		return 0, fmt.Errorf("failed write, session is not open")
	}
	return client.remoteStdin.Write(p)
}

// WriteClose well, writeCloser for client
func (client *RemoteClient) WriteClose() error {
	if client.remoteStdin == nil {
		return fmt.Errorf("failed close, session is not open")
	}
	return client.remoteStdin.Close()
}

// Signal process command signals
func (client *RemoteClient) Signal(sig os.Signal) error {
	if !client.sessOpened {
		return fmt.Errorf("session is not open")
	}

	switch sig {
	case os.Interrupt:
		// TODO: Turns out that .Signal(ssh.SIGHUP) doesn't work for me.
		// Instead, sending \x03 to the remote session works for me,
		// which sounds like something that should be fixed/resolved
		// upstream in the golang.org/x/crypto/ssh pkg.
		// https://github.com/golang/go/issues/4115#issuecomment-66070418
		if client.remoteStdin == nil {
			return fmt.Errorf("failed write signal, session is not open")
		}

		client.remoteStdin.Write([]byte("\x03"))
		return client.sess.Signal(ssh.SIGINT)
	default:
		return fmt.Errorf("%v not supported", sig)
	}
}

// -----------------------------
// scp part
// -----------------------------

// func removePortFromHostname(c *RemoteClient) {
// 	if c == nil {
// 		log.Panic("24A502B0-ADC9-4AFD-86E7-8DDE04E0F732: c is nil")
// 	}

// 	l := kemba.New("sshclient::removePortFromHostname").Printf

// 	l("port checking")
// 	if strings.Contains(c.Host, ":") {
// 		l("trimming host port")
// 		c.Host = strings.Split(c.Host, ":")[0]
// 	}
// }

// buildRemoteCommand constructs the command string to be run on the remote host.
func (client *RemoteClient) buildRemoteCommand(task entity.Task) string {
	l := kemba.New("SSHClient.build_remote_command").Printf

	command := task.Run
	isSudo := task.Sudo
	sudoPassword := client.Password
	Env := client.Env

	finalEnv := ns.InjectNamespacesAndEnvs(task, *Env, client)
	command = sudo.FormatCommandBasedOnSudo(isSudo, sudoPassword, finalEnv, command, client, task, false)

	l("done building command")
	return command
}

// ListFiles возвращает список всех файлов в директории dir, исключая .DS_Store
func ListFiles(dir string) ([]string, error) {
	l := kemba.New("SSHClient.ListFiles").Printf
	var files []string
	l("inspecting paths is dist folder")
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			l("error while walking through directories: %s", err)
			return err
		}

		// Пропускаем саму директорию
		if path == dir {
			l("skipping root directory: %s", path)
			return nil
		}

		// Пропускаем файлы .DS_Store
		if info.Name() == ".DS_Store" {
			l("skipping .DS_Store file: %s", path)
			return nil
		}

		// Добавляем только файлы, не директории
		if !info.IsDir() {
			l("adding file: %s")
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		l("error while walking through directories: %s", err)
		return nil, err
	}

	l("done inspecting ./dist")
	return files, nil
}

// FindAllPaths находит все файлы и папки в указанной директории
// и возвращает их пути в виде среза строк
func FindAllPaths(rootPath string) ([]string, error) {
	l := kemba.New("SSHClient.FindAllPaths").Printf
	var paths []string

	l("inspecting paths is dist folder")
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {

			return err
		}

		if strings.Contains(path, ".DS_Store") {

			return nil
		}

		if path == rootPath { // Skip the root directory itself

			return nil
		}

		return nil
	})

	if err != nil {

		return nil, err
	}

	return paths, nil
}
