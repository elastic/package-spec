// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"net/http"
	"path"
	"strconv"

	ve "github.com/elastic/package-spec/code/go/internal/errors"

	"github.com/elastic/package-spec/code/go/internal/validator/semantic"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	"github.com/rakyll/statik/fs"

	// Loads package specs as an in-memory filesystem
	_ "github.com/elastic/package-spec/code/go/internal/spec"
)

// Spec represents a package specification
type Spec struct {
	version  semver.Version
	fs       http.FileSystem
	specPath string
}

type validationRules []func(pkgRoot string) ve.ValidationErrors

// NewSpec creates a new Spec for the given version
func NewSpec(version semver.Version) (*Spec, error) {
	majorVersion := strconv.FormatUint(version.Major(), 10)

	bundle, err := fs.New()
	if err != nil {
		return nil, errors.Wrap(err, "could not load specifications")
	}

	specPath := "/" + majorVersion
	if _, err := bundle.Open(specPath); err != nil {
		return nil, errors.Wrapf(err, "could not load specification for version [%s]", version.String())
	}

	s := Spec{
		version,
		bundle,
		specPath,
	}

	return &s, nil
}

// ValidatePackage validates the given Package against the Spec
func (s Spec) ValidatePackage(pkg Package) ve.ValidationErrors {
	var errs ve.ValidationErrors

	rootSpecPath := path.Join(s.specPath, "spec.yml")
	rootSpec, err := newFolderSpec(s.fs, rootSpecPath)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "could not read root folder spec file"))
		return errs
	}

	// Syntactic validations
	errs = rootSpec.validate(pkg.Name, pkg.RootPath)
	if len(errs) != 0 {
		return errs
	}

	// Semantic validations
	rules := validationRules{
		semantic.ValidateKibanaObjectIDs,
		semantic.ValidateVersionIntegrity,
		semantic.ValidateFieldGroups,
	}
	return rules.validate(pkg.RootPath)
}

func (vr validationRules) validate(pkgRoot string) ve.ValidationErrors {
	var errs ve.ValidationErrors
	for _, validationRule := range vr {
		err := validationRule(pkgRoot)
		errs.Append(err)
	}

	return errs
}
