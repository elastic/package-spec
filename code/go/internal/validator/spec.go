// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"fmt"
	"io/fs"
	"path"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	spec "github.com/elastic/package-spec"
	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/fspath"
	"github.com/elastic/package-spec/code/go/internal/validator/semantic"
)

// Spec represents a package specification
type Spec struct {
	version semver.Version
	fs      fs.FS
}

type validationRule func(pkg fspath.FS) ve.ValidationErrors

type validationRules []validationRule

const maxVersion = "2.0.0"

// NewSpec creates a new Spec for the given version
func NewSpec(version semver.Version) (*Spec, error) {
	// TODO: Check version with the changelog.
	if version.GreaterThan(semver.MustParse(maxVersion)) {
		return nil, fmt.Errorf("could not load specification for version [%s]", version.String())
	}

	s := Spec{
		version,
		spec.FS(),
	}

	return &s, nil
}

// ValidatePackage validates the given Package against the Spec
func (s Spec) ValidatePackage(pkg Package) ve.ValidationErrors {
	var errs ve.ValidationErrors

	var rootSpec folderSpec
	rootSpecPath := path.Join(pkg.Type, "spec.yml")
	err := rootSpec.load(s.fs, rootSpecPath)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "could not read root folder spec file"))
		return errs
	}

	// Syntactic validations
	errs = rootSpec.validate(&pkg, ".")
	if len(errs) != 0 {
		return errs
	}

	// Semantic validations
	return s.rules(rootSpec).validate(&pkg)
}

func (s Spec) rules(rootSpec folderSpec) validationRules {
	rulesDef := []struct {
		fn    validationRule
		since *semver.Version
		until *semver.Version
	}{
		{fn: semantic.ValidateKibanaObjectIDs},
		{fn: semantic.ValidateVersionIntegrity},
		{fn: semantic.ValidateChangelogLinks},
		{fn: semantic.ValidatePrerelease},
		{fn: semantic.ValidateFieldGroups},
		{fn: semantic.ValidateFieldsLimits(rootSpec.Limits.FieldsPerDataStreamLimit)},
		{fn: semantic.ValidateUniqueFields, since: semver.MustParse("2.0.0")},
		{fn: semantic.ValidateDimensionFields},
		{fn: semantic.ValidateRequiredFields},
	}

	var validationRules validationRules
	for _, rule := range rulesDef {
		if rule.since != nil && s.version.LessThan(rule.since) {
			continue
		}
		if rule.until != nil && !s.version.LessThan(rule.until) {
			continue
		}
		validationRules = append(validationRules, rule.fn)
	}

	return validationRules
}

func (vr validationRules) validate(fsys fspath.FS) ve.ValidationErrors {
	var errs ve.ValidationErrors
	for _, validationRule := range vr {
		err := validationRule(fsys)
		errs.Append(err)
	}

	return errs
}
