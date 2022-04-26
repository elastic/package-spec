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
	"github.com/elastic/package-spec/code/go/internal/fspath"
	"github.com/elastic/package-spec/code/go/internal/validator/semantic"
)

// Spec represents a package specification
type Spec struct {
	version  semver.Version
	fs       fs.FS
	specPath string
}

type validationRules []func(pkg fspath.FS) ve.ValidationErrors

// NewSpec creates a new Spec for the given version
func NewSpec(version semver.Version) (*Spec, error) {
	majorVersion := strconv.FormatUint(version.Major(), 10)

	specFS := spec.FS()
	specPath := majorVersion
	if _, err := specFS.Open(specPath); err != nil {
		return nil, errors.Wrapf(err, "could not load specification for version [%s]", version.String())
	}

	s := Spec{
		version,
		specFS,
		specPath,
	}

	return &s, nil
}

// ValidatePackage validates the given Package against the Spec
func (s Spec) ValidatePackage(pkg Package) ve.ValidationErrors {
	var errs ve.ValidationErrors

	var rootSpec folderSpec
	rootSpecPath := path.Join(s.specPath, "spec.yml")
	err := rootSpec.load(s.fs, rootSpecPath)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "could not read root folder spec file"))
		return errs
	}

	// Syntactic validations
	errs = rootSpec.validate(pkg.Name, &pkg, ".")
	if len(errs) != 0 {
		return errs
	}

	// Semantic validations
	rules := validationRules{
		semantic.ValidateKibanaObjectIDs,
		semantic.ValidateVersionIntegrity,
		semantic.ValidateTopChangelogLink,
		semantic.ValidatePrerelease,
		semantic.ValidateFieldGroups,
		semantic.ValidateFieldsLimits(rootSpec.Limits.FieldsPerDataStreamLimit),
		semantic.ValidateUniqueFields,
		semantic.ValidateDimensionFields,
		semantic.ValidateRequiredFields,
	}

	return rules.validate(&pkg)
}

func (vr validationRules) validate(fsys fspath.FS) ve.ValidationErrors {
	var errs ve.ValidationErrors
	for _, validationRule := range vr {
		err := validationRule(fsys)
		errs.Append(err)
	}

	return errs
}
