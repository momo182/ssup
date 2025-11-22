package localhost

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/clok/kemba"
	"github.com/goforj/godump"
	"github.com/gookit/goutil/fsutil"
	"github.com/mitchellh/go-homedir"
	"github.com/momo182/ssup/src/entity"
	ns "github.com/momo182/ssup/src/gateway/namespace"
	shellcheckService "github.com/momo182/ssup/src/gateway/shellcheck"
	sudo "github.com/momo182/ssup/src/gateway/sudo_formatter"
	"github.com/samber/oops"
	"golang.org/x/crypto/ssh"
)

// LocalhostClient is a wrapper over the SSH connection/sessions.
type LocalhostClient struct {
	cmd       *exec.Cmd
	user      string
	stdin     io.WriteCloser
	Host      string
	stdout    io.Reader
	stderr    io.Reader
	running   bool
	Env       *entity.EnvList //export FOO="bar"; export BAR="baz";
	nameSpace string
	Inventory *entity.Inventory
	// encryptedPassword []byte
	// encryption        string
}

func (client *LocalhostClient) String() string {
	return fmt.Sprintf("local connection: %s, tube: %s", client.Host, client.nameSpace)
}

// GetHost returns the host of the LocalhostClient.
func (client *LocalhostClient) GetHost() string {
	return client.Host
}

// GetTube is a method to get the tube name of the LocalhostClient.
//
// Returns:
//   - The name of the tube as a string.
func (client LocalhostClient) GetTube() string {

	return client.nameSpace
}

// SetTube sets the tube name for the LocalhostClient.
//
// Args:
//   - name (string): The new tube name to set.
func (client *LocalhostClient) SetTube(name string) {

	client.nameSpace = name
}

// GetSSHConfig returns a pointer to an SSH client configuration struct.
//
// Returns:
//   - A pointer to the SSH client config. Note that for local machines,
//     this method always returns nil, as it doesn't need to reconnect.
func (client *LocalhostClient) GetSSHConfig() *ssh.ClientConfig {
	// simple passthru, local machine wont have to reconnect

	return nil // nolint: unused
}

// GetConnection acquires a connection with the remote host using SSH.
//
// It is assumed that this method will only be called on the localhost client,
// and thus does not require an existing connection.
func (client LocalhostClient) GetConnection() *ssh.Client {
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
func (client *LocalhostClient) GetShell() string { // FuncName: GetShell
	if client.Inventory.Bash {
		return "bash"
	}

	if client.Inventory.Sh {
		return "sh"
	}

	return ""
}

// GetInventory returns the inventory of the LocalhostClient.
func (client *LocalhostClient) GetInventory() *entity.Inventory { // FuncName: GetInventory
	return client.Inventory
}

// Connect connects to the host using SSH.
//
// Args:
//   - _ entity.NetworkHost: The network host to connect to.
//
// Returns:
//   - An error if there's a problem connecting, otherwise nil.
func (client *LocalhostClient) Connect(_ entity.NetworkHost) error { // FuncName: Connect
	u, err := user.Current()
	inventory, err := GatherInventory(nil)
	if err != nil {
		return err
	}

	if !inventory.Bash && !inventory.Sh {
		return errors.New("bash or sh not found")
	}

	inventory.User = u.Username
	client.Inventory = inventory

	// // do dist copy here
	// err = copyDistFolders(c)
	// if err != nil {
	// 	return fmt.Errorf("failed to copy dist folders for local client")
	// }
	return nil
}

// copyDistFolders this is actually a copy from ssh package
// but it doesnot have to be, we could skip the copy altogether
// and just set PATH to include the folder
func (client *LocalhostClient) CopyDistFolders() error {
	l := kemba.New("gw::ssh::LocalClient.copyDistFolders").Printf
	l("copying dist folder")

	if client == nil {
		return errors.New("client is nil")
	}

	if client.Inventory == nil {
		return errors.New("client inventory is nil")
	}

	l(godump.DumpStr(client.Inventory))

	// check if dist of ssupdist is present and if so copy it
	// to remote host ~/.local/ssup/dist
	currentDir, err := os.Getwd()
	if err != nil {
		return errors.New("failed to get current directory")
	}

	srcPath := filepath.Join(currentDir, "dist")

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

// Run starts the given task on localhost, does not wait for it to finish.
func (client *LocalhostClient) Run(task *entity.Task) error { // FuncName: Run
	l := kemba.New("gateway::localhost::Run").Printf

	l(fmt.Sprintf("Running task: %s", godump.DumpStr(task.Env)))
	if client.running {
		return fmt.Errorf("Command already running")
	}

	// if task.Env len is != 0 append those
	// to c.Env
	if len(task.Env.Keys()) != 0 {
		// c.Env = append(c.Env, task.Env...)
		for _, key := range task.Env.Keys() {
			client.Env.Set(key, task.Env.Get(key))
		}
	}

	// oldInvocation := []string{"bash", "-c", c.Env.AsExport()+task.Run}
	newInvocation := client.buildLocalCommand(*task)
	l("dump: B1DA9A9A-452B-4739-A8D8-15AEEA0D658D")
	l(godump.DumpStr(newInvocation))
	cmd := exec.Command(client.GetShell(), "-c", newInvocation)
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

	client.stdout = outPipe
	client.stderr = errPipe
	client.stdin = inPipe
	l("prepping tty")

	if task.TTY {
		client.stdin = inPipe
	}

	l("all pipes are set up")
	client.cmd = cmd

	l("prepared command: %s", cmd.String())
	l("^^^ this should have vars exported !!!!")

	l("as for the pipes...")

	l("starting local party")
	if err := client.cmd.Start(); err != nil {
		l("sometimes even local party can fail")
		return entity.ErrTask{Task: task, Reason: err.Error()}
	}

	client.running = true
	return nil
}

// Wait waits until the command finishes.
func (client *LocalhostClient) Wait() error {
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

	if !client.running {
		return fmt.Errorf("Trying to wait on stopped command")
	}
	err = client.cmd.Wait()
	if err != nil {
		oops.Trace("B4730A7E-1E1A-478B-A499-51FEB83BCD88").
			Hint("waiting for command").
			Wrap(err)
	}
	client.running = false

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
		client.Host = "localhost"
		ns.Namespaces.SetFromEnvString(string(envlines), client.Host)
		data := ns.Namespaces.Get(client.Host)
		l("data:\n%s", godump.DumpStr(data))
	}

	return err
}

// Close closes the client.
func (client *LocalhostClient) Close() error {
	return nil
}

// Stdin returns the client's standard input.
func (client *LocalhostClient) Stdin() io.WriteCloser {
	return client.stdin
}

// Stderr returns the client's standard error.
func (client *LocalhostClient) Stderr() io.Reader {
	return client.stderr
}

// Stdout returns the client's standard output.
func (client *LocalhostClient) Stdout() io.Reader {
	return client.stdout
}

// Prefix returns the client's prefix.
func (client *LocalhostClient) Prefix() (string, int) {
	user := os.Getenv("LOGNAME")
	host := user + "@localhost" + " | "
	return entity.ResetColor + host, len(host)
}

// Write implements writing byte into stdin.
func (client *LocalhostClient) Write(p []byte) (n int, err error) {
	if client.stdin == nil {
		return 0, fmt.Errorf("Trying to write to closed stdin")
	}
	return client.stdin.Write(p)
}

func (client *LocalhostClient) WriteClose() error {
	if client.stdin == nil {
		return fmt.Errorf("Trying to close to closed stdin")
	}
	return client.stdin.Close()
}

func (client *LocalhostClient) Signal(sig os.Signal) error {
	return client.cmd.Process.Signal(sig)
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

func (client *LocalhostClient) Upload(src, dst string) error {
	l := kemba.New("gw::local::upload").Printf
	src, err := homedir.Expand(src)
	if err != nil {
		return err
	}
	dst, err = homedir.Expand(dst)
	if err != nil {
		return err
	}
	l("kinda uploading %s to %s", src, dst)
	l("but its just a copy operation")

	return client.copyPath(src, dst, l)
}

func (client *LocalhostClient) Download(src, dst string, silent bool, _ bool) error {
	l := kemba.New("gw::local::download").Printf
	src, err := homedir.Expand(src)
	if err != nil {
		return err
	}
	dst, err = homedir.Expand(dst)
	if err != nil {
		return err
	}
	l("kinda downloading %s to %s", src, dst)
	l("but its just a copy operation")

	return client.copyPath(src, dst, l)
}

// copyPath handles the actual file/directory copying logic
func (client *LocalhostClient) copyPath(src, dst string, l func(format string, args ...interface{})) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		l("failed to stat source: %v", err)
		return err
	}

	if srcInfo.IsDir() {
		return client.copyDirectory(src, dst, l)
	}
	return client.copyFile(src, dst, l)
}

// copyFile copies a single file from src to dst
func (client *LocalhostClient) copyFile(src, dst string, l func(format string, args ...interface{})) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		l("failed to open source file: %v", err)
		return err
	}
	defer sourceFile.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		l("failed to create destination directory: %v", err)
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		l("failed to create destination file: %v", err)
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		l("failed to copy file content: %v", err)
		return err
	}

	// Preserve file mode
	srcInfo, err := os.Stat(src)
	if err == nil {
		if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
			l("failed to preserve file mode: %v", err)
			// Non-fatal error, continue
		}
	}

	l("successfully copied file %s to %s", src, dst)
	return nil
}

// copyDirectory recursively copies a directory from src to dst
func (client *LocalhostClient) copyDirectory(src, dst string, l func(format string, args ...interface{})) error {
	// Create destination directory
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		l("failed to create destination directory: %v", err)
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		l("failed to read source directory: %v", err)
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := client.copyDirectory(srcPath, dstPath, l); err != nil {
				return err
			}
		} else {
			if err := client.copyFile(srcPath, dstPath, l); err != nil {
				return err
			}
		}
	}

	l("successfully copied directory %s to %s", src, dst)
	return nil
}

func (client *LocalhostClient) GenerateOnRemote(data []byte, dest string) error {
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

	debugData := godump.DumpStr(map[string]any{
		"src":  len(data),
		"dest": dest,
	})

	l(debugData)

	// if /System/Library is directory
	// check if dest contains /home/username and replace it with home from UserHomeDir
	_, e = os.Stat("/System/Library")

	if e == nil {
		l("/System/Library is dir")
		dest = strings.Replace(dest, "/home/"+client.user, home, 1)
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
func (client *LocalhostClient) buildLocalCommand(task entity.Task) string {
	l := kemba.New("LocalhostClient.build_remote_command").Printf

	command := task.Run
	isSudo := task.Sudo
	sudoPassword := ""
	Env := client.Env

	finalEnv := ns.InjectNamespacesAndEnvs(task, *Env, client)
	command = sudo.FormatCommandBasedOnSudo(isSudo, sudoPassword, finalEnv, command, client, task, true)

	l("done building command")
	return command
}
