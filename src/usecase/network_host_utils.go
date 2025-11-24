package usecase

import (
	"bufio"
	"io"
	"strings"
	"unicode"

	"github.com/gookit/goutil/strutil"
)

// This file now only contains functions that might be used internally in the usecase package
// The main parsing functions were moved to the utils subpackage to break circular dependencies

func findAndStepOverKey(r int, input, key string) int {
	if strutil.ContainsOne(input, []string{key}) {
		r = strings.Index(input, key)
		r = r + len(key)
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
