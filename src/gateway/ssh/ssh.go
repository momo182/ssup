package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bitfield/script"
	"github.com/clok/kemba"
	"github.com/davecgh/go-spew/spew"
	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/fsutil"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/lobby"
	svc "github.com/momo182/ssup/src/lobby"
	spass "github.com/momo182/ssup/src/shared/checksshpass"
	"github.com/samber/oops"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHClient is a wrapper over the SSH connection/sessions.
type SSHClient struct {
	conn         *ssh.Client
	sess         *ssh.Session
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
	RcloneCfg    string
	tube         string
}

func init() {

}

// ErrConnect describes connection error
type ErrConnect struct {
	User   string
	Host   string
	Reason string
}

func (e ErrConnect) Error() string {
	return fmt.Sprintf(`Connect("%v@%v"): %v`, e.User, e.Host, e.Reason)
}

func (c *SSHClient) GetHost() string {
	return c.Host
}

func (c SSHClient) GetTube() string {
	return c.tube
}

func (c *SSHClient) SetTube(name string) {
	c.tube = name
}

func (c *SSHClient) SetRcloneCfg(config string) {
	c.RcloneCfg = config
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
	svc.Lobby.KeyAuth = &auth
	l("done initializing SSH authentication method")
}

// SSHDialFunc can dial an ssh server and return a client
type SSHDialFunc func(net, addr string, config *ssh.ClientConfig) (*ssh.Client, error)

// Connect creates SSH connection to a specified host.
// It expects the host of the form "[ssh://]host[:port]".
func (c *SSHClient) Connect(host entity.NetworkHost) error {
	return c.ConnectWith(host, ssh.Dial)
}

// ConnectWith creates a SSH connection to a specified host. It will use dialer to establish the
// connection.
// TODO: Split Signers to its own method.
func (c *SSHClient) ConnectWith(host entity.NetworkHost, dialer SSHDialFunc) error {
	l := kemba.New("gateway::ssh::SSHClient.ConnectWith").Printf
	l("connecting to %v", host)

	if c.connOpened {
		return fmt.Errorf("Already connected")
	}

	var authMethods []ssh.AuthMethod
	initAuthMethodOnce.Do(initAuthMethod)
	l("checking password auth")
	authMethods = spass.CheckPasswordAuth(authMethods, host)

	err := c.parseHost(host.Host)
	if err != nil {
		return err
	}

	l("creating config")
	config := &ssh.ClientConfig{
		User:            c.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	l("creating ssh client")
	c.conn, err = dialer("tcp", c.Host, config)
	if err != nil {
		return ErrConnect{c.User, c.Host, err.Error()}
	}
	c.connOpened = true
	l("done creating ssh client")

	// add namespace for the host
	l("adding namespace for host: %v", c.Host)
	svc.Lobby.Namespaces.Add(c.Host)

	return nil
}

// Run runs the task.Run command remotely on c.host.
func (c *SSHClient) Run(task *entity.Task) error {
	l := kemba.New("gateway::ssh::Run").Printf
	//nil check
	l("negative programming checks")
	if task == nil {
		return errors.New("got nil task")
	}

	if c.running {
		return fmt.Errorf("Session already running")
	}
	if c.sessOpened {
		return fmt.Errorf("Session already connected")
	}

	// if task.Env len is != 0 append those
	// to c.Env
	l("append envs to client")
	if len(task.Env.Keys()) != 0 {
		// c.Env = append(c.Env, task.Env...)
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
	}

	// Start the remote command.
	command := c.buildRemoteCommand(*task)
	l("starting remote command: %v", command)
	if err := sess.Start(command); err != nil {
		return entity.ErrTask{Task: task, Reason: err.Error()}
	}

	c.sess = sess
	c.sessOpened = true
	c.running = true
	l("done with Run func")
	return nil
}

// Wait waits until the remote command finishes and exits.
// It closes the SSH session.
func (c *SSHClient) Wait() error {
	config := c.RcloneCfg
	l := kemba.New("gateway::ssh::SSHClient.Wait").Printf
	if c == nil {
		log.Panic("7782C3CF-5E8E-4740-9F7E-D68A9B2ED71C: no ssh client passed")
	}

	if !c.running {
		return fmt.Errorf("Trying to wait on stopped session")
	}

	err := c.sess.Wait()
	c.sess.Close()
	c.running = false
	c.sessOpened = false

	// this code is a copy from `func (c *SSHClient) Download()`

	uuid, e := uuid.GenerateUUID()
	if e != nil {
		return oops.Trace("1352DBFA-8126-4A53-BEFF-4EC04A6B61E1").
			Hint("generating uuid").
			Wrap(e)
	}
	remoteName := "ssup_remote-" + uuid

	l("check if rclone is available")
	rclone := lobby.MustFindRclone()

	// check if host has port and cut it
	removePortFromHostname(c)

	l("create rclone config: %v", config)
	e = createRcloneConfig(rclone, remoteName, c, config)
	if e != nil {
		return oops.Trace("FFDD911D-997C-4BA2-89DF-7E2303088B10").
			Hint("create config to upload to remote").
			With("remote name", remoteName).
			Wrap(e)
	}

	envsPull := exec.Command("rclone", "--config", config, "cat", remoteName+":"+entity.VARS_TAIL)
	l("will run rclone command: %s", envsPull.String())
	envlines, e := envsPull.CombinedOutput()
	if e != nil {
		if envsPull.ProcessState.ExitCode() == 3 {
			l("ok to skip cat now")
		} else {
			return oops.Trace("FD08F4F9-EA36-4330-8B6B-908E272E6B7C").
				Hint("pulling envs from host").
				With("output", envlines).
				Wrap(e)
		}
	}
	l("output:\n%s", envlines)

	// TODO add namespace interaction here
	svc.Lobby.Namespaces.SetFromEnvString(string(envlines), c.Host)
	data := svc.Lobby.Namespaces.Get(c.Host)
	l("data:\n%s", dump.Format(data))

	envsDrop := exec.Command("rclone", "--config", config, "deletefile", remoteName+":"+entity.VARS_TAIL)
	l("will run rclone command: %s", envsDrop.String())
	o, e := envsDrop.CombinedOutput()
	if e != nil {
		if envsPull.ProcessState.ExitCode() == 3 {
			l("ok to skip dropping remote env storage now")
		} else {
			return oops.Trace("DE417902-4D50-4004-8182-711679A63259").
				Hint("dropping envs from host").
				With("output", o).
				Wrap(e)
		}
	}

	l("delete remote")
	destroyRcloneConfig := exec.Command(rclone, "--config", config, "config", "delete", remoteName)
	_, e = destroyRcloneConfig.CombinedOutput()
	if e != nil {
		l("failed to run command: %v", e)
		return e
	}

	// TODO remove rclone config here too
	return err
}

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

// Upload local file to remote server
func (c *SSHClient) Upload(localPath, remotePath string, config string) error {
	l := kemba.New("sshclient.Upload").Printf
	var rcloneCopyCmd = "copyto"

	uuid, e := uuid.GenerateUUID()
	if e != nil {
		return oops.Trace("1352DBFA-8126-4A53-BEFF-4EC04A6B61E1").
			Hint("generating uuid").
			Wrap(e)
	}
	remoteName := "ssup_remote-" + uuid

	l("check if rclone is available")
	rclone := lobby.MustFindRclone()

	// check if host has port and cut it
	removePortFromHostname(c)

	l("create rclone config")
	e = createRcloneConfig(rclone, remoteName, c, config)
	if e != nil {
		return oops.Trace("FFDD911D-997C-4BA2-89DF-7E2303088B10").
			Hint("create config to upload to remote").
			With("remote name", remoteName).
			Wrap(e)
	}

	realPath := fsutil.ResolvePath(localPath)
	// check if realPath is a directory
	if fsutil.IsDir(realPath) {
		rcloneCopyCmd = "copy"
	}

	l("prepare copy command")
	// check if localPath is a directory
	var copyCommand *exec.Cmd
	switch fsutil.IsDir(localPath) {
	case true:
		copyCommand = exec.Command(rclone, "--config", config, "--exclude", ".git/", "-P", rcloneCopyCmd, localPath, remoteName+":"+remotePath)
	default:
		copyCommand = exec.Command(rclone, "--config", config, "-P", rcloneCopyCmd, localPath, remoteName+":"+remotePath)
	}

	l("public run command: %v", copyCommand)
	copyCommand.Stdout = os.Stdout
	copyCommand.Stderr = os.Stderr

	e = copyCommand.Start()
	if e != nil {
		l("failed to run command: %v", e)
		return e
	}

	e = copyCommand.Wait()
	if e != nil {
		l("failed to wait for command: %v", e)
		return e
	}

	return nil
}

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

// Download file from remote
func (c *SSHClient) Download(remotePath, localPath string, silent bool) error {
	config := c.RcloneCfg
	l := kemba.New("sshclient.Download").Printf
	remoteName := "remote"

	is_rclone := script.Exec("sh -c 'which rclone'").ExitStatus() == 0
	if !is_rclone {
		fmt.Println("Please install rclone on your system, and make it available in $PATH")
		os.Exit(131)
	}

	// check if host has port and cut it
	if strings.Contains(c.Host, ":") {
		l("trimming host port")
		c.Host = strings.Split(c.Host, ":")[0]
	}

	initRcloneCmd := exec.Command("rclone", "--config", config, "config", "create", remoteName, "sftp", "host", c.Host, "user", c.User, "pass", c.Password)
	_, e := initRcloneCmd.CombinedOutput()
	if e != nil {
		l("failed to run command: %v", e)
		return e

	}

	copyCommand := exec.Command("rclone", "--config", config, "-P", "copy", remoteName+":"+remotePath, localPath)
	if !silent {
		copyCommand.Stdout = os.Stdout
		copyCommand.Stderr = os.Stderr
	}
	e = copyCommand.Start()
	if e != nil {
		l("failed to run command: %v", e)
		return e
	}

	e = copyCommand.Wait()
	if e != nil {
		l("failed to wait for command: %v", e)
		return e
	}

	destroyRcloneConfig := exec.Command("rclone", "--config", config, "config", "delete", remoteName)
	_, e = destroyRcloneConfig.CombinedOutput()
	if e != nil {
		l("failed to run command: %v", e)
		return e

	}
	return nil
}

// GenerateOnRemote basically cats file content to "~/" + entity.TASK_TAIL on remote
func (c *SSHClient) GenerateOnRemote(data []byte) error {
	l := kemba.New("sshclient.GenerateOnRemote").Printf
	config := c.RcloneCfg
	oldCmd := string(data)
	data = []byte(lobby.RegisterCmd + oldCmd)
	l("processing:\ndump: FC693B9D-DA60-4DA9-B783-647270E27BBC\n%s", string(lobby.Lobby.Shellcheck.AddNumbers(data)))

	uuid, e := uuid.GenerateUUID()
	if e != nil {
		return oops.Trace("1352DBFA-8126-4A53-BEFF-4EC04A6B61E1").
			Hint("generating uuid").
			Wrap(e)
	}
	remoteName := "ssup_sudo_remote-" + uuid

	dest := entity.TASK_TAIL
	debugData := spew.Sdump(map[string]any{
		"src":  data,
		"dest": dest,
	})
	l(debugData)

	l("check if rclone is available")
	rclone := lobby.MustFindRclone()

	// check if host has port and cut it
	removePortFromHostname(c)

	l("create rclone config")
	e = createRcloneConfig(rclone, remoteName, c, config)
	if e != nil {
		return oops.Trace("9ED9976F-9C69-4017-83AF-744AC40F2B9A").
			Hint("create config to generate on remote").
			With("remote name", remoteName).
			Wrap(e)
	}

	l("prepare rcat command")
	copyCommand := exec.Command(rclone, "--config", config, "rcat", remoteName+":"+entity.TASK_TAIL)
	copyCommand.Stdin = bytes.NewReader(data)
	l(fmt.Sprintf("copy:\n    src: %s\n    dest: %s\n", "user data", dest))

	l("silent run command: %v", copyCommand)
	o, e := copyCommand.CombinedOutput()
	if e != nil {
		if strings.Contains(string(o), "sftp: \"Failure\" (SSH_FX_FAILURE)") {
			l("no space left on device: %v", e)
			return e
		}

		l("failed to run command: %v", e)
		return e
	}

	l("delete remote")
	destroyRcloneConfig := exec.Command(rclone, "--config", config, "config", "delete", remoteName)
	_, e = destroyRcloneConfig.CombinedOutput()
	if e != nil {
		l("failed to run command: %v", e)
		return e

	}

	return nil
}

func createRcloneConfig(rclone string, remoteName string, c *SSHClient, config string) error {
	if c == nil {
		log.Panic("10EE087A-D3DA-4F16-A1D2-F71E11DE9EAD: c is nil")
	}

	l := kemba.New("sshclient::createRcloneConfig").Printf

	initRcloneCmd := exec.Command(rclone, "--config", config, "config", "create", remoteName, "sftp", "host", c.Host, "user", c.User, "pass", c.Password)
	o, e := initRcloneCmd.CombinedOutput()
	if e != nil {
		l("failed to run command: %v", e)
		return oops.Trace("68545C6A-62E1-446D-95A8-817EBA27390A").
			Hint("creating rclone config").
			With("CombinedOutput", o).
			Wrap(e)
	}
	return nil
}

// buildRemoteCommand constructs the command string to be run on the remote host.
func (c *SSHClient) buildRemoteCommand(task entity.Task) string {
	l := kemba.New("SSHClient.build_remote_command").Printf

	command := lobby.RegisterCmd + task.Run
	sudo := task.Sudo
	scriptName := entity.TASK_TAIL
	exportCmd := "export"
	sudoPassword := c.Password
	Env := c.Env

	finalEnv := lobby.InjectNamespacesAndEnvs(task, *Env, c)
	command = lobby.FormatCommandBasedOnSudo(sudo, sudoPassword, finalEnv, exportCmd, scriptName, command, c, task)

	l("done building command")
	return command
}
