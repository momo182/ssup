package localhost

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"

	"github.com/bitfield/script"
	"github.com/clok/kemba"
	"github.com/davecgh/go-spew/spew"
	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/fsutil"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/lobby"
	"github.com/samber/oops"
)

// Client is a wrapper over the SSH connection/sessions.
type LocalhostClient struct {
	cmd       *exec.Cmd
	user      string
	stdin     io.WriteCloser
	Host      string
	stdout    io.Reader
	stderr    io.Reader
	running   bool
	Env       *entity.EnvList //export FOO="bar"; export BAR="baz";
	RcloneCfg string
	tube      string
}

func (c *LocalhostClient) GetHost() string {
	return c.Host
}

func (c LocalhostClient) GetTube() string {
	return c.tube
}

func (c *LocalhostClient) SetTube(name string) {
	c.tube = name
}

func (c *LocalhostClient) Connect(_ entity.NetworkHost) error {
	u, err := user.Current()
	if err != nil {
		return err
	}

	c.user = u.Username
	return nil
}

func (c *LocalhostClient) SetRcloneCfg(config string) {
	c.RcloneCfg = config
}

func (c *LocalhostClient) Run(task *entity.Task) error {
	l := kemba.New("gateway::localhost::Run").Printf
	var err error

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
	cmd := exec.Command("bash", "-c", newInvocation)
	// cmd.Stdin = bytes.NewReader([]byte(newInvocation))
	c.cmd = cmd

	l("prepared command: %s", cmd.String())
	l("^^^ this should have vars exported !!!!")

	l("as for the pipes...")
	c.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return err
	}

	c.stderr, err = cmd.StderrPipe()
	if err != nil {
		return err
	}

	c.stdin, err = cmd.StdinPipe()
	if err != nil {
		return err
	}
	l("all pipes are set up")

	l("starting local party")
	if err := c.cmd.Start(); err != nil {
		l("sometimes even local party can fail")
		return entity.ErrTask{Task: task, Reason: err.Error()}
	}

	c.running = true
	return nil
}

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

		tgt := home + "/" + entity.VARS_TAIL
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
	if _, err := os.Stat(home + "/" + entity.VARS_TAIL); errors.Is(err, fs.ErrNotExist) {
		l("no vars file found, skipping")
	} else {
		envsPull := exec.Command("cat", home+"/"+entity.VARS_TAIL)
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
		lobby.Lobby.Namespaces.SetFromEnvString(string(envlines), c.Host)
		data := lobby.Lobby.Namespaces.Get(c.Host)
		l("data:\n%s", dump.Format(data))
	}

	return err
}

func (c *LocalhostClient) Close() error {
	return nil
}

func (c *LocalhostClient) Stdin() io.WriteCloser {
	return c.stdin
}

func (c *LocalhostClient) Stderr() io.Reader {
	return c.stderr
}

func (c *LocalhostClient) Stdout() io.Reader {
	return c.stdout
}

func (c *LocalhostClient) Prefix() (string, int) {
	host := c.user + "@localhost" + " | "
	return entity.ResetColor + host, len(host)
}

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

func (c *LocalhostClient) Upload(src, dst string, config string) error {
	l := kemba.New("localclient.Upload").Printf

	is_rclone := script.Exec("sh -c 'which rclone'").ExitStatus() == 0
	if !is_rclone {
		fmt.Println("Please install rclone on your system, and make it available in $PATH")
		os.Exit(13)
	}

	var copyCommand *exec.Cmd
	switch fsutil.IsDir(src) {
	case true:
		copyCommand = exec.Command("rclone", "--config", config, "--exclude", ".git/", "-P", "copyto", src, dst)
	default:
		copyCommand = exec.Command("rclone", "--config", config, "-P", "copyto", src, dst)
	}

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
	l := kemba.New("localclient.Download").Printf
	config := c.RcloneCfg

	is_rclone := script.Exec("sh -c 'which rclone'").ExitStatus() == 0
	if !is_rclone {
		fmt.Println("Please install rclone on your system, and make it available in $PATH")
		os.Exit(122)
	}

	copyCommand := exec.Command("rclone", "--config", config, "-P", "copy", src, dst)
	if !silent {
		copyCommand.Stdout = os.Stdout
		copyCommand.Stderr = os.Stderr
	}

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

func (c *LocalhostClient) GenerateOnRemote(data []byte) error {
	l := kemba.New("sshclient.GenerateOnRemote").Printf
	l("processing:\ndump: 19E5FE65-20A8-4050-992E-F3FA5A7AFFCF\n%s", string(lobby.Lobby.Shellcheck.AddNumbers(data)))
	home, e := os.UserHomeDir()
	if e != nil {
		return oops.Trace("C603225E-0928-4613-A4AB-8E0CAE1C4D10").
			Hint("failed to get home directory").
			Wrap(e)
	}
	dest := "_ssup_exec_script.sh"
	debugData := spew.Sdump(map[string]any{
		"src":  data,
		"dest": dest,
	})
	l(debugData)

	l("check if rclone is available")
	rclone := lobby.MustFindRclone()

	l(fmt.Sprintf("copy:\n    src: %s\n    dest: %s\n", "user data", dest))
	l("prepare rcat command")
	copyCommand := exec.Command(rclone, "--config", c.RcloneCfg, "rcat", home+"/"+entity.TASK_TAIL)
	copyCommand.Stdin = bytes.NewReader(data)
	l(fmt.Sprintf("copy:\n    src: %s\n    dest: %s\n", "user data", dest))
	return nil
}

// buildRemoteCommand constructs the command string to be run on the remote host.
func (c *LocalhostClient) buildLocalCommand(task entity.Task) string {
	l := kemba.New("LocalhostClient.build_remote_command").Printf

	command := lobby.RegisterCmd + task.Run
	sudo := task.Sudo
	scriptName := entity.TASK_TAIL
	exportCmd := "export"
	sudoPassword := ""
	Env := c.Env

	finalEnv := lobby.InjectNamespacesAndEnvs(task, *Env, c)
	command = lobby.FormatCommandBasedOnSudo(sudo, sudoPassword, finalEnv, exportCmd, scriptName, command, c, task)
	l("done building command")
	return command
}
