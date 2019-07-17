// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package engine

import (
	"bytes"
	"testing"
)

func TestWriteWorkdir(t *testing.T) {
	buf := new(bytes.Buffer)
	writeWorkdir(buf, "/tmp/drone-temp")

	want := "cd /tmp/drone-temp\n"
	if got := buf.String(); got != want {
		t.Errorf("Want workding dir %q, got %q", want, got)
	}
}

func TestWriteSecrets(t *testing.T) {
	buf := new(bytes.Buffer)
	sec := []*Secret{{Env: "a", Data: []byte("b")}}
	writeSecrets(buf, "linux", sec)

	want := "export a=\"b\"\n"
	if got := buf.String(); got != want {
		t.Errorf("Want secret script %q, got %q", want, got)
	}

	buf.Reset()
	writeSecrets(buf, "windows", sec)
	want = "$Env:a = \"b\"\n"
	if got := buf.String(); got != want {
		t.Errorf("Want secret script %q, got %q", want, got)
	}
}

func TestWriteEnv(t *testing.T) {
	buf := new(bytes.Buffer)
	env := map[string]string{"a": "b", "c": "d"}
	writeEnviron(buf, "linux", env)

	want := "export a=\"b\"\nexport c=\"d\"\n"
	if got := buf.String(); got != want {
		t.Errorf("Want environment script %q, got %q", want, got)
	}

	buf.Reset()
	writeEnviron(buf, "windows", env)
	want = "$Env:a = \"b\"\n$Env:c = \"d\"\n"
	if got := buf.String(); got != want {
		t.Errorf("Want environment script %q, got %q", want, got)
	}
}

func TestRemoveCommand(t *testing.T) {
	got := removeCommand("linux", "/tmp/drone-temp")
	want := "rm -rf /tmp/drone-temp"
	if got != want {
		t.Errorf("Want rm script %q, got %q", want, got)
	}

	got = removeCommand("windows", `C:\Windows\Temp\Drone-temp`)
	want = `powershell -noprofile -noninteractive -command "Remove-Item C:\Windows\Temp\Drone-temp -Recurse -Force"`
	if got != want {
		t.Errorf("Want rm script %q, got %q", want, got)
	}
}
