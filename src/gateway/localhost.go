package gateway

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"

	"github.com/bitfield/script"
	"github.com/clok/kemba"
	"github.com/davecgh/go-spew/spew"
	"github.com/momo182/ssup/src/entity"
	"github.com/samber/oops"
)

// Client is a wrapper over the SSH connection/sessions.
type LocalhostClient struct {
	cmd     *exec.Cmd
	user    string
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
	running bool
	Env     string //export FOO="bar"; export BAR="baz";
}

func (c *LocalhostClient) Connect(_ entity.NetworkHost) error {
	u, err := user.Current()
	if err != nil {
		return err
	}

	c.user = u.Username
	return nil
}

func (c *LocalhostClient) Run(task *entity.Task) error {
	var err error

	if c.running {
		return fmt.Errorf("Command already running")
	}

	cmd := exec.Command("bash", "-c", c.Env+task.Run)
	c.cmd = cmd

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

	if err := c.cmd.Start(); err != nil {
		return entity.ErrTask{Task: task, Reason: err.Error()}
	}

	c.running = true
	return nil
}

func (c *LocalhostClient) Wait() error {
	if !c.running {
		return fmt.Errorf("Trying to wait on stopped command")
	}
	err := c.cmd.Wait()
	c.running = false
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

func (c *LocalhostClient) Upload(src, dst string) error {
	l := kemba.New("sshclient.Upload").Printf

	is_rclone := script.Exec("sh -c 'which rclone'").ExitStatus() == 0
	if !is_rclone {
		fmt.Println("Please install rclone on your system, and make it available in $PATH")
		os.Exit(1)
	}

	copyCommand := exec.Command("rclone", "-P", "copyto", src, dst)
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
	l := kemba.New("sshclient.Upload").Printf

	is_rclone := script.Exec("sh -c 'which rclone'").ExitStatus() == 0
	if !is_rclone {
		fmt.Println("Please install rclone on your system, and make it available in $PATH")
		os.Exit(1)
	}

	copyCommand := exec.Command("rclone", "-P", "copy", src, dst)
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
	l("processing:\ndump: 19E5FE65-20A8-4050-992E-F3FA5A7AFFCF\n%s", string(addNumbers(data)))
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
	rclone := mustFindRclone()

	l(fmt.Sprintf("copy:\n    src: %s\n    dest: %s\n", "user data", dest))
	l("prepare rcat command")
	copyCommand := exec.Command(rclone, "rcat", home+"/"+entity.TASK_TAIL)
	copyCommand.Stdin = bytes.NewReader(data)
	l(fmt.Sprintf("copy:\n    src: %s\n    dest: %s\n", "user data", dest))
	return nil
}
