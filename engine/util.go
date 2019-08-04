// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package engine

import (
	"fmt"
	"io"
	"sort"
)

// helper function writes a shell command to the io.Writer that
// changes the current working directory.
func writeWorkdir(w io.Writer, path string) {
	fmt.Fprintf(w, "cd %s", path)
	fmt.Fprintln(w)
}

// helper function writes a shell command to the io.Writer that
// exports all secrets as environment variables.
func writeSecrets(w io.Writer, os string, secrets []*Secret) {
	for _, s := range secrets {
		writeEnv(w, os, s.Env, string(s.Data))
	}
}

// helper function writes a shell command to the io.Writer that
// exports the key value pairs as environment variables.
func writeEnviron(w io.Writer, os string, envs map[string]string) {
	var keys []string
	for k := range envs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		writeEnv(w, os, k, envs[k])
	}
}

// helper function writes a shell command to the io.Writer that
// exports and key value pair as an environment variable.
func writeEnv(w io.Writer, os, key, value string) {
	switch os {
	case "windows":
		fmt.Fprintf(w, "$Env:%s = %q", key, value)
		fmt.Fprintln(w)
	default:
		fmt.Fprintf(w, "export %s=%q", key, value)
		fmt.Fprintln(w)
	}
}

// helper function returns a shell command for removing a
// directory that is compatible with the operating system.
func removeCommand(os, path string) string {
	switch os {
	case "windows":
		return fmt.Sprintf("powershell -noprofile -noninteractive -command \"Remove-Item %s -Recurse -Force\"", path)
	default:
		return fmt.Sprintf("rm -rf %s", path)
	}
}
