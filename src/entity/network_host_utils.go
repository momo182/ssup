package entity

import (
	"strings"

	"github.com/gookit/goutil/strutil"
)

func findPasswordStart(host string) int {
	r := -1
	r = findAndStepOverKey(r, host, PassSeparator)
	return r
}

func findPasswordEnd(host string) int {
	r := len(host)
	if strutil.ContainsOne(host, []string{TubeNameSeparator}) {
		r = strings.Index(host, TubeNameSeparator)
	}
	return r
}

func findTubeNameStart(host string) int {
	r := -1
	r = findAndStepOverKey(r, host, TubeNameSeparator)
	return r
}

func findTubeNameEnd(host string) int {
	r := len(host)
	return r
}

func findAndStepOverKey(r int, input, key string) int {
	if strutil.ContainsOne(input, []string{key}) {
		r = strings.Index(input, key)
		r = r + len(key)
	}
	return r
}
