package localhost

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/fsutil"
	"github.com/momo182/ssup/src/entity"
	shellcheckService "github.com/momo182/ssup/src/gateway/shellcheck"
	"github.com/momo182/ssup/src/lobby"
	"github.com/samber/oops"
	"golang.org/x/crypto/ssh"
)

// LocalhostClient is a wrapper over the SSH connection/sessions.
type LocalhostClient struct {
	cmd               *exec.Cmd
	user              string
	stdin             io.WriteCloser
	Host              string
	stdout            io.Reader
	stderr            io.Reader
	running           bool
	Env               *entity.EnvList //export FOO="bar"; export BAR="baz";
	tube              string
	Inventory         *entity.Inventory
	encryptedPassword []byte
	encryption        string
}

// GetHost returns the host of the LocalhostClient.
func (c *LocalhostClient) GetHost() string {
	return c.Host
}

// GetTube is a method to get the tube name of the LocalhostClient.
//
// Returns:
//   - The name of the tube as a string.
func (c LocalhostClient) GetTube() string {

	return c.tube
}

// SetTube sets the tube name for the LocalhostClient.
//
// Args:
//   - name (string): The new tube name to set.
func (c *LocalhostClient) SetTube(name string) {

	c.tube = name
}

// GetSSHConfig returns a pointer to an SSH client configuration struct.
//
// Returns:
//   - A pointer to the SSH client config. Note that for local machines,
//     this method always returns nil, as it doesn't need to reconnect.
func (c *LocalhostClient) GetSSHConfig() *ssh.ClientConfig {
	// simple passthru, local machine wont have to reconnect

	return nil // nolint: unused
}

// GetConnection acquires a connection with the remote host using SSH.
//
// It is assumed that this method will only be called on the localhost client,
// and thus does not require an existing connection.
func (c LocalhostClient) GetConnection() *ssh.Client {
	// simple passthru, local machine wont have to reconnect
	return nil
}

// SetConnection sets a pre-existing SSH connection for the current session.
//
// This is primarily used when reconnecting to an existing host.
func (c LocalhostClient) SetConnection(client *ssh.Client) {
	return
}

// GetShell returns the shell of the LocalhostClient.
func (c *LocalhostClient) GetShell() string { // FuncName: GetShell
	if c.Inventory.Bash {
		return "bash"
	}

	if c.Inventory.Sh {
		return "sh"
	}

	return ""
}

// GetInventory returns the inventory of the LocalhostClient.
func (c *LocalhostClient) GetInventory() *entity.Inventory { // FuncName: GetInventory
	return c.Inventory
}

// Connect connects to the host using SSH.
//
// Args:
//   - _ entity.NetworkHost: The network host to connect to.
//
// Returns:
//   - An error if there's a problem connecting, otherwise nil.
func (c *LocalhostClient) Connect(_ entity.NetworkHost) error { // FuncName: Connect
	u, err := user.Current()
	inventory, err := GatherInventory(nil)
	if err != nil {
		return err
	}

	if !inventory.Bash && !inventory.Sh {
		return errors.New("bash or sh not found")
	}

	inventory.User = u.Username
	c.Inventory = inventory
	return nil
}

// Run starts the given task on localhost, does not wait for it to finish.
func (c *LocalhostClient) Run(task *entity.Task) error { // FuncName: Run
	l := kemba.New("gateway::localhost::Run").Printf

	l(fmt.Sprintf("Running task: %s", dump.Format(task.Env)))
	if c.running {
		return fmt.Errorf("Command already running")
	}

	// if task.Env len is != 0 append those
	// to c.Env
	if len(task.Env.Keys()) != 0 {
		// c.Env = append(c.Env, task.Env...)
		for _, key := range task.Env.Keys() {
			c.Env.Set(key, task.Env.Get(key))
		}
	}

	// oldInvocation := []string{"bash", "-c", c.Env.AsExport()+task.Run}
	newInvocation := c.buildLocalCommand(*task)
	l("dump: B1DA9A9A-452B-4739-A8D8-15AEEA0D658D")
	l(dump.Format(newInvocation))
	cmd := exec.Command(c.GetShell(), "-c", newInvocation)
	// cmd.Stdin = bytes.NewReader([]byte(newInvocation))
	inPipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	c.stdout = outPipe
	c.stderr = errPipe
	c.stdin = inPipe
	l("prepping tty")

	if task.TTY {
		c.stdin = inPipe
	}

	l("all pipes are set up")
	c.cmd = cmd

	l("prepared command: %s", cmd.String())
	l("^^^ this should have vars exported !!!!")

	l("as for the pipes...")

	l("starting local party")
	if err := c.cmd.Start(); err != nil {
		l("sometimes even local party can fail")
		return entity.ErrTask{Task: task, Reason: err.Error()}
	}

	c.running = true
	return nil
}

// Wait waits until the command finishes.
func (c *LocalhostClient) Wait() error {
	l := kemba.New("localhost::Wait").Printf
	home, err := os.UserHomeDir()
	if err != nil {
		oops.Trace("FBF3ADC1-871C-492C-A53B-51765E0473A6").
			Hint("getting home dir").
			With("home", home).
			Wrap(err)
	}

	defer func() {

		tgt := home + "/" + entity.VARS_FILE
		l("cleanup: %s", tgt)
		// no error handling here is OKayish for the moment
		os.Remove(tgt)
	}()

	if !c.running {
		return fmt.Errorf("Trying to wait on stopped command")
	}
	err = c.cmd.Wait()
	if err != nil {
		oops.Trace("B4730A7E-1E1A-478B-A499-51FEB83BCD88").
			Hint("waiting for command").
			Wrap(err)
	}
	c.running = false

	l("grab vars from the run")
	// if home+"/"+entity.VARS_TAIL exists, read it, else skip
	if _, err := os.Stat(home + "/" + entity.VARS_FILE); errors.Is(err, fs.ErrNotExist) {
		l("no vars file found, skipping")
	} else {
		envsPull := exec.Command("cat", home+"/"+entity.VARS_FILE)
		l("will run command to pull envs: %s", envsPull.Args)
		envlines, e := envsPull.CombinedOutput()
		if e != nil {
			return oops.Trace("96827050-82E3-4CB3-B6D2-DE5EA1FA48C2").
				Hint("pulling envs from localhost").
				With("output", envlines).
				Wrap(e)
		}

		l("output:\n%s", envlines)
		c.Host = "localhost"
		lobby.ServiceRegistry.Namespaces.SetFromEnvString(string(envlines), c.Host)
		data := lobby.ServiceRegistry.Namespaces.Get(c.Host)
		l("data:\n%s", dump.Format(data))
	}

	return err
}

// Close closes the client.
func (c *LocalhostClient) Close() error {
	return nil
}

// Stdin returns the client's standard input.
func (c *LocalhostClient) Stdin() io.WriteCloser {
	return c.stdin
}

// Stderr returns the client's standard error.
func (c *LocalhostClient) Stderr() io.Reader {
	return c.stderr
}

// Stdout returns the client's standard output.
func (c *LocalhostClient) Stdout() io.Reader {
	return c.stdout
}

// Prefix returns the client's prefix.
func (c *LocalhostClient) Prefix() (string, int) {
	host := c.user + "@localhost" + " | "
	return entity.ResetColor + host, len(host)
}

// Write implements writing byte into stdin.
func (c *LocalhostClient) Write(p []byte) (n int, err error) {
	if c.stdin == nil {
		return 0, fmt.Errorf("Trying to write to closed stdin")
	}
	return c.stdin.Write(p)
}

func (c *LocalhostClient) WriteClose() error {
	if c.stdin == nil {
		return fmt.Errorf("Trying to close to closed stdin")
	}
	return c.stdin.Close()
}

func (c *LocalhostClient) Signal(sig os.Signal) error {
	return c.cmd.Process.Signal(sig)
}

// func (c *LocalhostClient) Upload(localPath, remotePath string, silent bool) error {
// 	l := kemba.New("localhost.Upload").Printf
// 	l("Uploading %s to %s", localPath, remotePath)
// 	e := fsutil.CopyFile(localPath, remotePath)
// 	if e != nil {
// 		return e
// 	}
// 	return nil
// }

func (c *LocalhostClient) Upload(src, dst string) error {
	l := kemba.New("gw::local::upload").Printf
	l("kinda uploading %s to %s", src, dst)
	l("but its just a cp call")

	var copyCommand *exec.Cmd
	copyCommand = exec.Command("cp", "-R", src, dst)

	copyCommand.Stdout = os.Stdout
	copyCommand.Stderr = os.Stderr

	e := copyCommand.Start()
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

func (c *LocalhostClient) Download(src, dst string, silent bool) error {
	l := kemba.New("gw::local::download").Printf
	l("kinda downloading %s to %s", src, dst)
	l("but its just a cp call")

	var copyCommand *exec.Cmd
	copyCommand = exec.Command("cp", "-R", src, dst)

	copyCommand.Stdout = os.Stdout
	copyCommand.Stderr = os.Stderr

	e := copyCommand.Start()
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

func (c *LocalhostClient) GenerateOnRemote(data []byte, dest string) error {
	l := kemba.New("gw::local::GenerateOnRemote").Printf
	var shellcheck entity.ShellCheckFacade
	shellcheck = &shellcheckService.ShellCheckProvider{}
	l("processing:\ndump: 19E5FE65-20A8-4050-992E-F3FA5A7AFFCF\n%s", string(shellcheck.AddNumbers(data)))
	home, e := os.UserHomeDir()
	if e != nil {
		return oops.Trace("C603225E-0928-4613-A4AB-8E0CAE1C4D10").
			Hint("failed to get home directory").
			Wrap(e)
	}

	debugData := dump.Format(map[string]any{
		"src":  len(data),
		"dest": dest,
	})

	l(debugData)

	// if /System/Library is directory
	// check if dest contains /home/username and replace it with home from UserHomeDir
	SysLibHandle, e := os.Stat("/System/Library")
	if e != nil {
		return oops.Trace("6B937FB1-90E2-43A9-A152-1D218AF8A352").
			Hint("failed to check if /System/Library is directory").
			Wrap(e)
	}

	if SysLibHandle.IsDir() {
		l("/System/Library is dir")
		dest = strings.Replace(dest, "/home/"+c.user, home, 1)
	}

	// write the data to a destination file
	destPath := dest
	l("writing to %s", destPath)

	// check if destPath exists
	// if it doesn't exist, create it
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		path := fsutil.Dir(destPath)
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return oops.Trace("3F19551C-9627-4967-8894-524589090420").
				Hint("failed to create destination directory").
				With("dest", path).
				Wrap(err)
		}
	}

	e = os.WriteFile(destPath, data, 0755)
	if e != nil {
		return oops.Trace("08829609-6F76-4825-9185-477470705316").
			Hint("failed to write file").
			Wrap(e)
	}
	return nil
}

// buildRemoteCommand constructs the command string to be run on the remote host.
func (c *LocalhostClient) buildLocalCommand(task entity.Task) string {
	l := kemba.New("LocalhostClient.build_remote_command").Printf

	command := task.Run
	sudo := task.Sudo
	sudoPassword := ""
	Env := c.Env

	finalEnv := lobby.InjectNamespacesAndEnvs(task, *Env, c)
	command = lobby.FormatCommandBasedOnSudo(sudo, sudoPassword, finalEnv, command, c, task, true)

	l("done building command")
	return command
}
