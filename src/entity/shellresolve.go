package entity

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"github.com/clok/kemba"
	"github.com/gookit/goutil/dump"
	"github.com/samber/oops"
)

func ResolveShell(value string) (string, error) {
	l := kemba.New("shellresolve").Printf
	// remove the prefix
	value = value[2:] // remove the `$(`
	// remove the suffix
	value = value[:len(value)-1] // remove the `)`

	// and run this as shell command
	l("about to run command %q", value)
	cmd := exec.Command("sh", "-c", value)
	cmd.Stderr = os.Stderr
	out, e := cmd.Output()
	if e != nil {
		return "", oops.
			Trace("6928F3B4-0D17-45FB-9633-DABA63E163A1").
			Hint("failed to run command").
			With("cmd", value).
			Wrap(e)
	}

	// limit value to only printable characters
	outReader := bytes.NewBuffer(out)
	clean, e := FilterNonPrintable(outReader)
	if e != nil {
		return "", oops.
			Trace("CE16720F-D992-4EA3-9E68-3F1A740A66C1").
			Hint("failed to filer non-printable characters").
			With("outReader", outReader).
			Wrap(e)
	}

	l("cmd dump:\n%s\nvalue: %s", dump.Format(cmd), clean)
	return clean, nil
}

func IsShell(cmd string) bool {
	var r bool
	// reverse cmd string
	if strings.HasPrefix(cmd, "$(") && strings.HasSuffix(cmd, ")") {
		r = true
	}
	return r
}

// @this original 81A0C135-69EA-4736-AFD4-1D132A2DB91E@
func FilterNonPrintable(r io.Reader) (string, error) {
	reader := bufio.NewReader(r)
	var buffer []byte
	for {
		b, _, err := reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				break // Завершаем чтение, если достигнут конец файла.
			}
			return "", err
		}
		if unicode.IsPrint(rune(b)) {
			buffer = append(buffer, []byte(string(b))...)
		}
	}
	return string(buffer), nil
}
