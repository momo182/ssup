package usecase

import (
	"fmt"
	"os"
	"strings"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
)

func GenerateSUPENVFrom(cliVars entity.EnvList, vars entity.EnvList) {
	// SUP_ENV is generated only from CLI env vars.
	// Separate loop to omit duplicates.
	supEnv := ""
	for _, v := range cliVars {
		supEnv += fmt.Sprintf(" -e %v=%q", v.Key, v.Value)
	}
	vars.Set("SUP_ENV", strings.TrimSpace(supEnv))
}

// SetEnvValues sets two (entity.EnvList)s with the values of the environment variables
// from initialArgs.EnvVars and returns one (entity.EnvList)
func SetEnvValues(vars entity.EnvList, initialArgs *entity.InitialArgs) entity.EnvList {
	// Create a new list of environment variables cliVars of type entity.EnvList
	var cliVars entity.EnvList

	// Iterate over the list of environment variables in initialArgs
	for _, env := range initialArgs.EnvVars {
		// If the environment variable is empty, skip it and move to the next one
		if len(env) == 0 {
			continue
		}

		// Find the index of the '=' character in the environment variable
		i := strings.Index(env, "=")

		// If the '=' character is not found, add the environment variable to the vars list with an empty value
		if i < 0 {
			if len(env) > 0 {
				vars.Set(env, "")
			}
			continue
		}

		// If the '=' character is found, split the environment variable into key and value by the '=' character and add them to the vars and cliVars lists
		vars.Set(env[:i], env[i+1:])
		cliVars.Set(env[:i], env[i+1:])
	}

	// Return the cliVars list
	return cliVars
}

// MergeVars merges two entity.EnvList lists, one from supfile and one from network,
// resolves the values of the environment variables with shell substitution, and
// returns the merged list of environment variables
func MergeVars(conf *entity.Supfile, network *entity.Network) entity.EnvList {
	l := kemba.New("usecase > MergeVars").Printf
	// Create a new list of environment variables vars of type entity.EnvList
	var vars entity.EnvList

	// Iterate over the list of environment variables in conf and network
	l("looping over merged env vars")
	for _, val := range append(conf.Env, network.Env...) {
		// Add the environment variable to the vars list
		l("adding env var: %v = %v ", val.Key, val.Value)
		vars.Set(val.Key, val.Value)
	}

	// Resolve the values in the vars list
	l("resolving env vars via shell")
	if err := ResolveValues(&vars); err != nil {
		// If there is an error, print it to the standard error output and exit with code 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Return the vars list
	return vars
}
