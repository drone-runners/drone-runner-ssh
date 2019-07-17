// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package resource

import (
	"errors"

	"github.com/drone/runner-go/manifest"

	"github.com/buildkite/yaml"
)

func init() {
	manifest.Register(parse)
}

// parse parses the raw resource and returns an Exec pipeline.
func parse(r *manifest.RawResource) (manifest.Resource, bool, error) {
	if !match(r) {
		return nil, false, nil
	}
	out := new(Pipeline)
	err := yaml.Unmarshal(r.Data, out)
	if err != nil {
		return out, true, err
	}
	err = lint(out)
	return out, true, err
}

// match returns true if the resource matches the kind and type.
func match(r *manifest.RawResource) bool {
	return r.Kind == Kind && r.Type == Type
}

// lint returns an error if any pipeline values are invalid.
func lint(pipeline *Pipeline) error {
	// ensure server configuration provided.
	if pipeline.Server.Host.Value == "" && pipeline.Server.Host.Secret == "" {
		return errors.New("Linter: invalid or missing server host")
	}
	if pipeline.Server.User.Value == "" && pipeline.Server.User.Secret == "" {
		return errors.New("Linter: invalid or missing server user")
	}
	if pipeline.Server.Password.Value == "" && pipeline.Server.Password.Secret == "" &&
		pipeline.Server.SSHKey.Value == "" && pipeline.Server.SSHKey.Secret == "" {
		return errors.New("Linter: invalid or missing server password or ssh_key")
	}

	// ensure pipeline steps are not unique.
	names := map[string]struct{}{}
	for _, step := range pipeline.Steps {
		if step.Name == "" {
			return errors.New("Linter: invalid or missing step name")
		}
		if _, ok := names[step.Name]; ok {
			return errors.New("Linter: duplicate step name")
		}
		names[step.Name] = struct{}{}
	}
	return nil
}
