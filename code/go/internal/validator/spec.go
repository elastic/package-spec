// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"io/fs"
	"path"
	"strconv"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	spec "github.com/elastic/package-spec"
	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/specschema"
	"github.com/elastic/package-spec/code/go/internal/yamlschema"
)

// Spec represents a package specification
type Spec struct {
	version  semver.Version
	fs       fs.FS
	rules    validationRulesBuilder
	specPath string
}

// NewSpec creates a new Spec for the given version
func NewSpec(version semver.Version) (*Spec, error) {
	majorVersion := strconv.FormatUint(version.Major(), 10)

	specFS := spec.FS()
	specPath := majorVersion
	if _, err := specFS.Open(specPath); err != nil {
		return nil, errors.Wrapf(err, "could not load specification for version [%s]", version.String())
	}

	specRules, err := newRulesBuilder(version)
	if err != nil {
		return nil, err
	}

	s := Spec{
		version:  version,
		fs:       specFS,
		rules:    specRules,
		specPath: specPath,
	}

	return &s, nil
}

// ValidatePackage validates the given Package against the Spec
func (s Spec) ValidatePackage(pkg Package) ve.ValidationErrors {
	var errs ve.ValidationErrors

	fileSpecLoader := yamlschema.NewFileSchemaLoader()
	loader := specschema.NewFolderSpecLoader(s.fs, fileSpecLoader)

	rootSpecPath := path.Join(s.specPath, pkg.Type)
	rootSpec, err := loader.Load(rootSpecPath)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "could not read root folder spec file"))
		return errs
	}

	// Syntactic validations
	validator := newValidator(rootSpec, &pkg)
	errs = validator.Validate()
	if len(errs) != 0 {
		return errs
	}

	// Semantic validation
	if s.rules != nil {
		return s.rules(rootSpec).validate(&pkg)
	}

	return nil
}
