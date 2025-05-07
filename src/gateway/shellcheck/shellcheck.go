package shellcheck

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/dump"
	"github.com/momo182/ssup/src/entity"
	"github.com/samber/oops"
)

type ShellCheckProvider struct{}

func New() *ShellCheckProvider {
	return &ShellCheckProvider{}
}

// Check runs shellcheck with the contents of the
// local: or run: blocks
func (s *ShellCheckProvider) Check(cmd string, cmdName string) error {
	l := kemba.New("gw::shellcheck::check").Printf

	scPath, err := exec.LookPath("shellcheck")
	l("shellcheck path:", scPath)
	l("command:\n%s", dump.Format(cmd))

	if err == nil {
		check := []string{"shellcheck", "-f", "tty", "-Calways", "-e", "SC2148,SC2155,SC2001", "-"}
		scCommand := exec.Command(check[0], check[1:]...)
		scCommand.Stdin = bytes.NewReader([]byte(cmd))
		out, e := scCommand.CombinedOutput()
		l("exit code:", scCommand.ProcessState.ExitCode())

		// this is the dance
		// to stop the pesky empty at the start of the output
		if scCommand.ProcessState.ExitCode() != 0 {
			fmt.Print(entity.ResetColor)
			fmt.Println("SHELLCHECK > command_name: " + cmdName)
			fmt.Println(string(out))
		}

		if e != nil {
			return oops.
				Trace("5386853C-E58D-4DBE-99F9-EE23C1E2444E").
				Hint("running shellcheck").
				Wrap(e)
		}
	}
	return nil
}

// AddNumbers adds numbers to each line
func (s *ShellCheckProvider) AddNumbers(data []byte) []byte {
	var result []byte
	asStrings := strings.Split(string(data), "\n")
	for id, line := range asStrings {
		id += 1
		var byteLine []byte
		byteLine = append([]byte(fmt.Sprintf("%3.d: ", id)), []byte(line+"\n")...)
		result = append(result, byteLine...)
	}
	return result
}
