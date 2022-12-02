// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package spec

import (
	"io/fs"
	"log"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	spec "github.com/elastic/package-spec/v2"
	"github.com/elastic/package-spec/v2/code/go/internal/jsonschema"
	"github.com/elastic/package-spec/v2/code/go/internal/loader"
)

// Spec represents a package specification
type Spec struct {
	version semver.Version
	fs      fs.FS
}

// NewSpec creates a new Spec for the given version
func NewSpec(version semver.Version) (*Spec, error) {
	specVersion, err := spec.CheckVersion(version)
	if err != nil {
		return nil, errors.Wrapf(err, "could not load specification for version [%s]", version.String())
	}
	if specVersion.Prerelease() != "" {
		log.Printf("Warning: package using an unreleased version of the spec (%s)", specVersion)
	}

	s := Spec{
		version,
		spec.FS(),
	}

	return &s, nil
}

// RenderJSONSchema returns the JSON Schemas related to the itemPath for this spec and a given package type
func (s Spec) RenderJSONSchema(itemPath, pkgType string) (*jsonschema.RenderedJSONSchema, error) {
	rootSpec, err := loader.LoadSpec(s.fs, s.version, pkgType)
	if err != nil {
		return nil, err
	}

	return jsonschema.JSONSchema(rootSpec, itemPath)
}

// RenderAllJSONSchemas returns all the JSON Schemas for this package spec and a given package type
func (s Spec) RenderAllJSONSchemas(pkgType string) ([]jsonschema.RenderedJSONSchema, error) {
	rootSpec, err := loader.LoadSpec(s.fs, s.version, pkgType)
	if err != nil {
		return nil, err
	}

	return jsonschema.AllJSONSchemas(rootSpec)
}
