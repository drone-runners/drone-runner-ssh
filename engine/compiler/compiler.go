// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package compiler

import (
	"context"
	"fmt"
	"strings"

	"github.com/drone-runners/drone-runner-ssh/engine"
	"github.com/drone-runners/drone-runner-ssh/runtime"

	"github.com/drone/runner-go/clone"
	"github.com/drone/runner-go/environ"
	"github.com/drone/runner-go/environ/provider"
	"github.com/drone/runner-go/manifest"
	"github.com/drone/runner-go/secret"

	"github.com/dchest/uniuri"
	"github.com/gosimple/slug"
)

// random generator function
var random = uniuri.New

// Compiler compiles the Yaml configuration file to an
// intermediate representation optimized for simple execution.
type Compiler struct {
	// Environ provides a set of environment variables that
	// should be added to each pipeline step by default.
	Environ provider.Provider

	// Secret returns a named secret value that can be injected
	// into the pipeline step.
	Secret secret.Provider
}

// Compile compiles the configuration file.
func (c *Compiler) Compile(ctx context.Context, args runtime.CompilerArgs) *engine.Spec {
	pipeline := args.Pipeline
	os := pipeline.Platform.OS

	spec := &engine.Spec{
		Platform: engine.Platform{
			OS:      pipeline.Platform.OS,
			Arch:    pipeline.Platform.Arch,
			Variant: pipeline.Platform.Variant,
			Version: pipeline.Platform.Version,
		},
		Server: engine.Server{
			Hostname: pipeline.Server.Host.Value,
			Username: pipeline.Server.User.Value,
			Password: pipeline.Server.Password.Value,
			SSHKey:   pipeline.Server.SSHKey.Value,
		},
	}

	// maybe load the server host variable from secret
	if s, ok := c.findSecret(ctx, args, pipeline.Server.Host.Secret); ok {
		spec.Server.Hostname = s
	}
	// maybe load the server username variable from secret
	if s, ok := c.findSecret(ctx, args, pipeline.Server.User.Secret); ok {
		spec.Server.Username = s
	}
	// maybe load the server password variable from secret
	if s, ok := c.findSecret(ctx, args, pipeline.Server.Password.Secret); ok {
		spec.Server.Password = s
	}
	// maybe load the server ssh_key variable from secret
	if s, ok := c.findSecret(ctx, args, pipeline.Server.SSHKey.Secret); ok {
		spec.Server.SSHKey = s
	}

	// append the port to the hostname if not exists
	if !strings.Contains(spec.Server.Hostname, ":") {
		spec.Server.Hostname = spec.Server.Hostname + ":22"
	}

	// create the root directory
	spec.Root = tempdir(os)

	// creates a home directory in the root.
	// note: mkdirall fails on windows so we need to create all
	// directories in the tree.
	homedir := join(os, spec.Root, "home", "drone")
	spec.Files = append(spec.Files, &engine.File{
		Path:  join(os, spec.Root, "home"),
		Mode:  0700,
		IsDir: true,
	})
	spec.Files = append(spec.Files, &engine.File{
		Path:  homedir,
		Mode:  0700,
		IsDir: true,
	})

	// creates a source directory in the root.
	// note: mkdirall fails on windows so we need to create all
	// directories in the tree.
	sourcedir := join(os, spec.Root, "drone", "src")
	spec.Files = append(spec.Files, &engine.File{
		Path:  join(os, spec.Root, "drone"),
		Mode:  0700,
		IsDir: true,
	})
	spec.Files = append(spec.Files, &engine.File{
		Path:  sourcedir,
		Mode:  0700,
		IsDir: true,
	})

	// creates the opt directory to hold all scripts.
	spec.Files = append(spec.Files, &engine.File{
		Path:  join(os, spec.Root, "opt"),
		Mode:  0700,
		IsDir: true,
	})

	// creates the netrc file
	if args.Netrc != nil && args.Netrc.Password != "" {
		netrcfile := getNetrc(os)
		netrcpath := join(os, homedir, netrcfile)
		netrcdata := fmt.Sprintf(
			"machine %s login %s password %s",
			args.Netrc.Machine,
			args.Netrc.Login,
			args.Netrc.Password,
		)
		spec.Files = append(spec.Files, &engine.File{
			Path: netrcpath,
			Mode: 0600,
			Data: []byte(netrcdata),
		})
	}

	// list the global environment variables
	globals, _ := c.Environ.List(ctx, &provider.Request{
		Build: args.Build,
		Repo:  args.Repo,
	})

	// create the default environment variables.
	envs := environ.Combine(
		provider.ToMap(
			provider.FilterUnmasked(globals),
		),
		args.Build.Params,
		environ.Proxy(),
		environ.System(args.System),
		environ.Repo(args.Repo),
		environ.Build(args.Build),
		environ.Stage(args.Stage),
		environ.Link(args.Repo, args.Build, args.System),
		clone.Environ(clone.Config{
			SkipVerify: pipeline.Clone.SkipVerify,
			Trace:      pipeline.Clone.Trace,
			User: clone.User{
				Name:  args.Build.AuthorName,
				Email: args.Build.AuthorEmail,
			},
		}),
		// TODO(bradrydzewski) windows variable HOMEDRIVE
		// TODO(bradrydzewski) windows variable LOCALAPPDATA
		map[string]string{
			"HOME":                homedir,
			"HOMEPATH":            homedir, // for windows
			"USERPROFILE":         homedir, // for windows
			"DRONE_HOME":          sourcedir,
			"DRONE_WORKSPACE":     sourcedir,
			"GIT_TERMINAL_PROMPT": "0",
		},
	)

	// create clone step, maybe
	if pipeline.Clone.Disable == false {
		clonepath := join(os, spec.Root, "opt", getExt(os, "clone"))
		clonefile := genScript(os,
			clone.Commands(
				clone.Args{
					Branch: args.Build.Target,
					Commit: args.Build.After,
					Ref:    args.Build.Ref,
					Remote: args.Repo.HTTPURL,
					Depth:  args.Pipeline.Clone.Depth,
				},
			),
		)

		cmd, args := getCommand(os, clonepath)
		spec.Steps = append(spec.Steps, &engine.Step{
			Name:      "clone",
			Args:      args,
			Command:   cmd,
			Envs:      envs,
			RunPolicy: engine.RunAlways,
			Files: []*engine.File{
				{
					Path: clonepath,
					Mode: 0700,
					Data: []byte(clonefile),
				},
			},
			Secrets:    []*engine.Secret{},
			WorkingDir: sourcedir,
		})
	}

	// create steps
	for _, src := range pipeline.Steps {
		buildslug := slug.Make(src.Name)
		buildpath := join(os, spec.Root, "opt", getExt(os, buildslug))
		buildfile := genScript(os, src.Commands)

		cmd, cmdArgs := getCommand(os, buildpath)
		dst := &engine.Step{
			Name:      src.Name,
			Args:      cmdArgs,
			Command:   cmd,
			Detach:    src.Detach,
			DependsOn: src.DependsOn,
			Envs: environ.Combine(envs,
				environ.Expand(
					convertStaticEnv(src.Environment),
				),
			),
			IgnoreErr:    strings.EqualFold(src.Failure, "ignore"),
			IgnoreStdout: false,
			IgnoreStderr: false,
			RunPolicy:    engine.RunOnSuccess,
			Files: []*engine.File{
				{
					Path: buildpath,
					Mode: 0700,
					Data: []byte(buildfile),
				},
			},
			Secrets:    convertSecretEnv(src.Environment),
			WorkingDir: sourcedir,
		}
		spec.Steps = append(spec.Steps, dst)

		// set the pipeline step run policy. steps run on
		// success by default, but may be optionally configured
		// to run on failure.
		if isRunAlways(src) {
			dst.RunPolicy = engine.RunAlways
		} else if isRunOnFailure(src) {
			dst.RunPolicy = engine.RunOnFailure
		}

		// if the pipeline step has unmet conditions the step is
		// automatically skipped.
		if !src.When.Match(manifest.Match{
			Action:   args.Build.Action,
			Cron:     args.Build.Cron,
			Ref:      args.Build.Ref,
			Repo:     args.Repo.Slug,
			Instance: args.System.Host,
			Target:   args.Build.Deploy,
			Event:    args.Build.Event,
			Branch:   args.Build.Target,
		}) {
			dst.RunPolicy = engine.RunNever
		}
	}

	if isGraph(spec) == false {
		configureSerial(spec)
	} else if pipeline.Clone.Disable == false {
		configureCloneDeps(spec)
	} else if pipeline.Clone.Disable == true {
		removeCloneDeps(spec)
	}

	// HACK: append masked global variables to secrets
	// this ensures the environment variable values are
	// masked when printed to the console.
	masked := provider.FilterMasked(globals)
	for _, step := range spec.Steps {
		for _, g := range masked {
			step.Secrets = append(step.Secrets, &engine.Secret{
				Name: g.Name,
				Data: []byte(g.Data),
				Mask: g.Mask,
				Env:  g.Name,
			})
		}
	}

	for _, step := range spec.Steps {
		for _, s := range step.Secrets {
			secret, ok := c.findSecret(ctx, args, s.Name)
			if ok {
				s.Data = []byte(secret)
			}
		}
	}

	return spec
}

// helper function attempts to find and return the named secret.
// from the secret provider.
func (c *Compiler) findSecret(ctx context.Context, args runtime.CompilerArgs, name string) (s string, ok bool) {
	if name == "" {
		return
	}

	// source secrets from the global secret provider
	// and the repository secret provider.
	provider := secret.Combine(
		args.Secret,
		c.Secret,
	)

	// TODO return an error to the caller if the provider
	// returns an error.
	found, _ := provider.Find(ctx, &secret.Request{
		Name:  name,
		Build: args.Build,
		Repo:  args.Repo,
		Conf:  args.Manifest,
	})
	if found == nil {
		return
	}
	return found.Data, true
}
