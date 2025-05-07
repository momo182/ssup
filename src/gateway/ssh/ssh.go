package ssh

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/fsutil"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/lobby"
	svc "github.com/momo182/ssup/src/lobby"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHClient is a wrapper over the SSH connection/sessions.
type SSHClient struct {
	conn              *ssh.Client
	sess              *ssh.Session
	ConnectOrder      entity.ConnectOrder
	User              string
	Host              string
	Password          string
	remoteStdin       io.WriteCloser
	remoteStdout      io.Reader
	remoteStderr      io.Reader
	connOpened        bool
	sessOpened        bool
	running           bool
	Env               *entity.EnvList
	Color             string
	tube              string
	Inventory         *entity.Inventory
	encryptedPassword []byte
	encryption        string
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

// GetHost returns the host of the SSHClient.
func (c *SSHClient) GetHost() string {
	return c.Host
}

// GetTube returns the tube of the SSHClient.
func (c SSHClient) GetTube() string {
	return c.tube
}

// GetConnection returns the client configuration of the SSHClient.
func (c SSHClient) GetSSHConfig() *ssh.ClientConfig {
	return c.ConnectOrder.ClientConfig
}

// GetConnection returns the client configuration of the SSHClient.
func (c SSHClient) GetConnection() *ssh.Client {
	return c.conn
}

// SetConnection sets the SSH client connection of the SSHClient.
func (c *SSHClient) SetConnection(client *ssh.Client) {
	c.conn = client
	c.connOpened = true
}

// SetTube sets the tube of the SSHClient.
func (c *SSHClient) SetTube(name string) {
	c.tube = name
}

// GetPassword returns the password of the SSHClient.
func (c *SSHClient) GetPassword() string {
	return c.Password
}

// SetPassword sets the password of the SSHClient.
func (c *SSHClient) SetPassword(pwd string) {
	c.Password = pwd
}

// GetEncryptedPassword returns the encrypted password for sudo.
func (c *SSHClient) GetEncryptedPassword() []byte {
	return c.encryptedPassword
}

// SetEncryptedPassword sets the encrypted password for sudo.
func (c *SSHClient) SetEncryptedPassword(pwd []byte) {
	c.encryptedPassword = pwd
}

// parseHost parses and normalizes <user>@<host:port> from a given string.
func (c *SSHClient) parseHost(host string) error {
	c.Host = host

	// Remove extra "ssh://" schema
	if len(c.Host) > 6 && c.Host[:6] == "ssh://" {
		c.Host = c.Host[6:]
	}

	// Split by the last "@", since there may be an "@" in the username.
	if at := strings.LastIndex(c.Host, "@"); at != -1 {
		c.User = c.Host[:at]
		c.Host = c.Host[at+1:]
	}

	// Add default user, if not set
	if c.User == "" {
		u, err := user.Current()
		if err != nil {
			return err
		}
		c.User = u.Username
	}

	if strings.Index(c.Host, "/") != -1 {
		return ErrConnect{c.User, c.Host, "unexpected slash in the host URL"}
	}

	// Add default port, if not set
	if strings.Index(c.Host, ":") == -1 {
		c.Host += ":22"
	}

	return nil
}

var initAuthMethodOnce sync.Once
var authMethods []ssh.AuthMethod

// initAuthMethod initiates SSH authentication method.
// initAuthMethod initiates SSH authentication method.
func initAuthMethod() {
	l := kemba.New("gateway::ssh::initAuthMethod").Printf
	l("initializing SSH authentication method")
	var signers []ssh.Signer

	// If there's a running SSH Agent, try to use its Private keys.
	l("check ssh agent is running")
	sock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err == nil {
		l("using SSH Agent")
		agent := agent.NewClient(sock)
		signers, _ = agent.Signers()
	}

	// Try to read user's SSH private keys from the standard paths.
	l("check if user has SSH private keys")
	files, _ := filepath.Glob(os.Getenv("HOME") + "/.ssh/id_*")
	for _, file := range files {
		if strings.HasSuffix(file, ".pub") {
			continue // Skip public keys.
		}
		data, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			continue
		}
		l("Using SSH private key: %s", file)
		signers = append(signers, signer)

	}
	l("found %v SSH signers", len(signers))
	auth := ssh.PublicKeys(signers...)
	svc.ServiceRegistry.KeyAuth = &auth
	l("done initializing SSH authentication method")
}

// SSHDialFunc can dial an ssh server and return a client
type SSHDialFunc func(net, addr string, config *ssh.ClientConfig) (*ssh.Client, error)

// Connect creates SSH connection to a specified host.
// It expects the host of the form "[ssh://]host[:port]".
func (c *SSHClient) Connect(host entity.NetworkHost) error {
	l := kemba.New("gw::ssh::SSHClient.Connect").Printf

	err := c.parseHost(host.Host)
	if err != nil {
		return err
	}

	var authMethods []ssh.AuthMethod
	initAuthMethodOnce.Do(initAuthMethod)

	l("checking password auth")
	authMethods = lobby.SetupAuthMethods(authMethods, host)

	l("creating config")
	l(dump.Format(host, authMethods))
	config := ssh.ClientConfig{
		User:            c.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	l(dump.Format(config))

	l("creating connect order")
	connectOrder := entity.ConnectOrder{
		Host:         host.Host,
		ClientConfig: &config,
	}
	l(dump.Format(connectOrder))

	c.ConnectOrder = connectOrder

	return c.ConnectWith(host, ssh.Dial)
}

// ConnectWith creates a SSH connection to a specified host. It will use dialer to establish the
// connection.
// TODO: Split Signers to its own method.
func (c *SSHClient) ConnectWith(host entity.NetworkHost, dialer SSHDialFunc) error {
	l := kemba.New("gw::ssh::SSHClient.ConnectWith").Printf
	l("connecting to %v", host)
	var err error

	if c.connOpened {
		return fmt.Errorf("Already connected")
	}
	config := c.ConnectOrder.ClientConfig
	if config == nil {
		return fmt.Errorf("ssh ClientConfig is nil")
	}

	l("creating ssh client")
	c.conn, err = dialer("tcp", c.Host, config)
	if err != nil {
		return ErrConnect{c.User, c.Host, err.Error()}
	}

	// TODO add inventory here
	inventory, err := GatherInventory(c.conn)
	if err != nil {
		return ErrInv{c.User, c.Host, err.Error()}
	}
	c.Inventory = inventory
	c.connOpened = true
	l("done creating ssh client")

	// add namespace for the host
	l("adding namespace for host: %v", c.Host)
	svc.ServiceRegistry.Namespaces.Add(c.Host)

	return nil
}

func isConnected(client *ssh.Client) bool {
	_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
	return err == nil
}

// Run runs the task.Run command remotely on c.host.
func (c *SSHClient) Run(task *entity.Task) error {
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
		// Request pseudo terminal
		if err := sess.RequestPty("xterm", 80, 40, modes); err != nil {
			return entity.ErrTask{Task: task, Reason: fmt.Sprintf("request for pseudo terminal failed: %s", err)}
		}
		l("tty requested")
	}

	// actuall create command from task
	command := c.buildRemoteCommand(*task)
	l("built following command:\n%s", dump.Format(command))

	if err := sess.Start(command); err != nil {
		return entity.ErrTask{Task: task, Reason: err.Error()}
	}

	c.sess = sess
	c.sessOpened = true
	c.running = true
	l("done with Run func")
	return nil
}

func (c *SSHClient) negativeChecksBeforeRun(task *entity.Task) error {
	if task == nil {
		return errors.New("got nil task")
	}

	if c.conn == nil {
		return errors.New("ssh client not connected")
	}

	if c.sess != nil {
		return errors.New("ssh session somehow... connected")
	}

	if c.running {
		return fmt.Errorf("Session already running")
	}

	if c.sessOpened {
		return fmt.Errorf("Session already connected")
	}
	return nil
}

func (c *SSHClient) GetShell() string {
	if c.Inventory.Bash {
		return "bash"
	}

	if c.Inventory.Sh {
		return "sh"
	}

	return ""
}

func (c *SSHClient) GetInventory() *entity.Inventory {
	return c.Inventory
}

// Wait waits until the remote command finishes and exits.
// It closes the SSH session.
func (c *SSHClient) Wait() error {
	l := kemba.New("gateway::ssh::SSHClient.Wait").Printf
	l("will wait for client: %s", c.Host)
	if c == nil {
		log.Panic("7782C3CF-5E8E-4740-9F7E-D68A9B2ED71C: no ssh client passed")
	}

	if !c.running {
		return fmt.Errorf("Trying to wait on stopped session")
	}

	e := c.sess.Wait()
	if e != nil {
		return entity.ErrTask{Task: nil, Reason: e.Error()}
	}
	c.sess.Close()
	c.running = false
	c.sessOpened = false

	return c.FetchEnvsWithTar()
}

func (c *SSHClient) FetchEnvsWithTar() error {
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
	defer client.Close()

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
	c.Download(remoteFilePath, localPath.Name(), false)
	remoteVars := fsutil.ReadAll(localPath)

	l("dump: 21D2239C-7DCD-4044-B30C-FFB5254EB0C2")
	l("%s", dump.Format(string(remoteVars)))

	svc.ServiceRegistry.Namespaces.SetFromEnvString(string(remoteVars), c.Host)
	debugData := svc.ServiceRegistry.Namespaces.Get(c.Host)
	l("data:\n%s", dump.Format(debugData))
	return nil
}

// func (c *SSHClient) FetchEnvsWithTar(config string, remoteName string) error {
// 	l := kemba.New("gw::ssh::SSHClient.FetchEnvsWithTar").Printf
// 	client := c.conn
// 	localPath, err := fsutil.TempDir("", "SSUP_TEMP_*.tmp")

// 	scpClient, err := scp.NewClientFromExistingSSH(client, &scp.ClientOption{})
// 	if err != nil {
// 		return oops.Trace("AE37AA9B-8E0A-49FB-91C6-D05C63C0637D").
// 			Hint("failed to create scp client").
// 			Wrap(err)

// 	}
// 	defer scpClient.Close()

// 	ctx := context.Background()
// 	fo := &scp.FileTransferOption{
// 		Context:      ctx,
// 		Timeout:      30 * time.Second,
// 		PreserveProp: true,
// 	}

// 	err = scpClient.CopyFileFromRemote(entity.VARS_FILE, localPath, fo)

// 	data, e := os.ReadFile(filepath.Join(localPath, entity.VARS_FILE))
// 	if e != nil {
// 		return oops.Trace("FB7B1D24-227C-44D1-8C55-5BCC57D41C73").
// 			Hint("failed to read file").
// 			With("localPath", localPath).
// 			Wrap(e)
// 	}
// 	externalEnvs := string(data)
// 	l("output:\n%s", externalEnvs)

// 	svc.Lobby.Namespaces.SetFromEnvString(string(externalEnvs), c.Host)
// 	debugData := svc.Lobby.Namespaces.Get(c.Host)
// 	l("data:\n%s", dump.Format(debugData))

// 	return nil
// }

// DialThrough will create a new connection from the ssh server sc is connected to. DialThrough is an SSHDialer.
func (c *SSHClient) DialThrough(net, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	conn, err := c.conn.Dial(net, addr)
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
func (c *SSHClient) Close() error {
	if c.sessOpened {
		c.sess.Close()
		c.sessOpened = false
	}
	if !c.connOpened {
		return fmt.Errorf("Trying to close the already closed connection")
	}

	err := c.conn.Close()
	c.connOpened = false
	c.running = false

	return err
}

// Stdin sets remote stdin
func (c *SSHClient) Stdin() io.WriteCloser {
	return c.remoteStdin
}

// Stderr sets remote stderr
func (c *SSHClient) Stderr() io.Reader {
	return c.remoteStderr
}

// Stdout sets remote stdout
func (c *SSHClient) Stdout() io.Reader {
	return c.remoteStdout
}

// Prefix sets prefix for printing
func (c *SSHClient) Prefix() (string, int) {
	l := kemba.New("gateway::ssh::SSHClient.Prefix").Printf
	hostName := c.Host
	if strings.Contains(c.Host, ":") {
		hostName = c.Host[:strings.Index(c.Host, ":")]
	}

	host := c.User + "@" + hostName + " | "
	l("host: %s", host)
	return c.Color + host + entity.ResetColor, len(host)
}

func (c *SSHClient) Write(p []byte) (n int, err error) {
	if c.remoteStdin == nil {
		return 0, fmt.Errorf("failed write, session is not open")
	}
	return c.remoteStdin.Write(p)
}

// WriteClose well, writeCloser for client
func (c *SSHClient) WriteClose() error {
	if c.remoteStdin == nil {
		return fmt.Errorf("failed close, session is not open")
	}
	return c.remoteStdin.Close()
}

// Signal process command signals
func (c *SSHClient) Signal(sig os.Signal) error {
	if !c.sessOpened {
		return fmt.Errorf("session is not open")
	}

	switch sig {
	case os.Interrupt:
		// TODO: Turns out that .Signal(ssh.SIGHUP) doesn't work for me.
		// Instead, sending \x03 to the remote session works for me,
		// which sounds like something that should be fixed/resolved
		// upstream in the golang.org/x/crypto/ssh pkg.
		// https://github.com/golang/go/issues/4115#issuecomment-66070418
		if c.remoteStdin == nil {
			return fmt.Errorf("failed write signal, session is not open")
		}

		c.remoteStdin.Write([]byte("\x03"))
		return c.sess.Signal(ssh.SIGINT)
	default:
		return fmt.Errorf("%v not supported", sig)
	}
}

// -----------------------------
// scp part
// -----------------------------

func removePortFromHostname(c *SSHClient) {
	if c == nil {
		log.Panic("24A502B0-ADC9-4AFD-86E7-8DDE04E0F732: c is nil")
	}

	l := kemba.New("sshclient::removePortFromHostname").Printf

	l("port checking")
	if strings.Contains(c.Host, ":") {
		l("trimming host port")
		c.Host = strings.Split(c.Host, ":")[0]
	}
}

// buildRemoteCommand constructs the command string to be run on the remote host.
func (c *SSHClient) buildRemoteCommand(task entity.Task) string {
	l := kemba.New("SSHClient.build_remote_command").Printf

	command := task.Run
	sudo := task.Sudo
	sudoPassword := c.Password
	Env := c.Env

	finalEnv := lobby.InjectNamespacesAndEnvs(task, *Env, c)
	command = lobby.FormatCommandBasedOnSudo(sudo, sudoPassword, finalEnv, command, c, task, false)

	l("done building command")
	return command
}
