// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"io/fs"
	"log"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	spec "github.com/elastic/package-spec/v2"
	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/loader"
	"github.com/elastic/package-spec/v2/code/go/internal/packages"
	"github.com/elastic/package-spec/v2/code/go/internal/spectypes"
	"github.com/elastic/package-spec/v2/code/go/internal/validator/semantic"
)

// Spec represents a package specification
type Spec struct {
	version semver.Version
	fs      fs.FS
}

type validationRule func(pkg fspath.FS) ve.ValidationErrors

type validationRules []validationRule

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

// ValidatePackage validates the given Package against the Spec
func (s Spec) ValidatePackage(pkg packages.Package) ve.ValidationErrors {
	var errs ve.ValidationErrors

	rootSpec, err := loader.LoadSpec(s.fs, s.version, pkg.Type)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "could not read root folder spec file"))
		return errs
	}

	// Syntactic validations
	validator := newValidator(rootSpec, &pkg)
	errs = append(errs, validator.Validate()...)

	// Semantic validations
	errs = append(errs, s.rules(pkg.Type, rootSpec).validate(&pkg)...)

	return processErrors(errs)
}

func substringInSlice(str string, list []string) bool {
	for _, substr := range list {
		if strings.Contains(str, substr) {
			return true
		}
	}
	return false
}

func processErrors(errs ve.ValidationErrors) ve.ValidationErrors {
	// Rename unclear error messages and filter out redundant errors
	var processedErrs ve.ValidationErrors
	msgTransforms := []struct {
		original string
		new      string
	}{
		{"Must not validate the schema (not)", "Must not be present"},
	}
	redundant := []string{
		"Must validate \"then\" as \"if\" was valid",
		"Must validate \"else\" as \"if\" was not valid",
		"Must validate all the schemas (allOf)",
		"Must validate at least one schema (anyOf)",
	}
	for _, e := range errs {
		for _, msg := range msgTransforms {
			if strings.Contains(e.Error(), msg.original) {
				processedErrs = append(processedErrs, errors.New(strings.Replace(e.Error(), msg.original, msg.new, 1)))
				continue
			}
			if substringInSlice(e.Error(), redundant) {
				continue
			}
			processedErrs = append(processedErrs, e)
		}
	}

	return processedErrs
}

func (s Spec) rules(pkgType string, rootSpec spectypes.ItemSpec) validationRules {
	rulesDef := []struct {
		fn    validationRule
		since *semver.Version
		until *semver.Version
		types []string
	}{
		{fn: semantic.ValidateVersionIntegrity},
		{fn: semantic.ValidateChangelogLinks},
		{fn: semantic.ValidatePrerelease},
		{fn: semantic.ValidateMinimumKibanaVersion},
		{fn: semantic.ValidateFieldGroups},
		{fn: semantic.ValidateFieldsLimits(rootSpec.MaxFieldsPerDataStream())},
		{fn: semantic.ValidateUniqueFields, since: semver.MustParse("2.0.0")},
		{fn: semantic.ValidateDimensionFields},
		{fn: semantic.ValidateDateFields},
		{fn: semantic.ValidateRequiredFields},
		{fn: semantic.ValidateExternalFieldsWithDevFolder},
		{fn: semantic.ValidateVisualizationsUsedByValue, types: []string{"integration"}},
		{fn: semantic.ValidateILMPolicyPresent, since: semver.MustParse("2.0.0"), types: []string{"integration"}},
		{fn: semantic.ValidateProfilingNonGA, types: []string{"integration"}},
		{fn: semantic.ValidateKibanaObjectIDs, types: []string{"integration"}},
	}

	var validationRules validationRules
	for _, rule := range rulesDef {
		if rule.since != nil && s.version.LessThan(rule.since) {
			continue
		}
		if rule.until != nil && !s.version.LessThan(rule.until) {
			continue
		}

		if rule.types != nil && !stringSliceContains(rule.types, pkgType) {
			continue
		}

		validationRules = append(validationRules, rule.fn)
	}

	return validationRules
}

func stringSliceContains(elems []string, v string) bool {
	for _, a := range elems {
		if a == v {
			return true
		}
	}
	return false
}

func (vr validationRules) validate(fsys fspath.FS) ve.ValidationErrors {
	var errs ve.ValidationErrors
	for _, validationRule := range vr {
		err := validationRule(fsys)
		errs.Append(err)
	}

	return errs
}
