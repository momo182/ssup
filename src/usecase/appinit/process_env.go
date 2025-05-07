package appinit

import (
	"fmt"
	"os"
	"strings"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
)

// GenerateSUPENVFrom generates the SUP_ENV environment variable
// from the environment variables in the vars list
func GenerateSUPENVFrom(vars *entity.EnvList) {
	supEnv := ""
	for _, key := range vars.Keys() {
		value := vars.Get(key)
		supEnv += fmt.Sprintf(" -e %v=%q", key, value)
	}
	vars.Set("SUP_ENV", strings.TrimSpace(supEnv))
}

// SetEnvValues sets vars from env pairs inside the initialArgs.EnvVars,
// values there will be given as 'key=value'
func SetEnvValues(vars *entity.EnvList, initialArgs *entity.InitialArgs) {
	l := kemba.New("usecase::SetEnvValues").Printf
	for _, env := range initialArgs.EnvVars {
		// If the environment variable is empty, skip it and move to the next one
		if len(env) == 0 {
			l("skipping empty env var")
			continue
		}

		// Find the index of the '=' character in the environment variable
		i := strings.Index(env, "=")
		l(fmt.Sprintf("index: %v, value: '%s'", i, env))

		// If the '=' character is not found, add the environment variable to the vars list with an empty value
		if i < 0 {
			l("skipping mailformed var (error: no '=' sign found): '%v'", env)
			continue
		}

		// If the '=' character is found, split the environment variable into key and value by the '=' character and add them to the vars and cliVars lists
		vars.Set(env[:i], env[i+1:])
	}
}

// MergeVars merges two entity.EnvList lists, one from supfile and one from network,
// resolves the values of the environment variables with shell substitution, and
// returns the merged list of environment variables
func MergeVars(conf *entity.Supfile, network *entity.Network) entity.EnvList {
	l := kemba.New("usecase::MergeVars").Printf
	var mergedVars entity.EnvList

	l("copyng from config")
	for _, key := range conf.Env.Keys() {
		value := conf.Env.Get(key)
		// Add the environment variable to the vars list
		l("adding conf env var: %v = %v ", key, value)
		mergedVars.Set(key, value)
	}

	for _, key := range network.Env.Keys() {
		value := network.Env.Get(key)
		// Add the environment variable to the vars list
		l("adding network env var: %v = %v ", key, value)
		mergedVars.Set(key, value)
	}

	// Resolve the values in the vars list
	l("resolving env vars via shell")
	if err := ResolveValues(&mergedVars); err != nil {
		// If there is an error, print it to the standard error output and exit with code 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(46)
	}

	// Return the vars list
	return mergedVars
}
