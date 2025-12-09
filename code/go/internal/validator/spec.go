// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"regexp"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"

	spec "github.com/elastic/package-spec/v3"
	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/loader"
	"github.com/elastic/package-spec/v3/code/go/internal/packages"
	"github.com/elastic/package-spec/v3/code/go/internal/spectypes"
	"github.com/elastic/package-spec/v3/code/go/internal/validator/semantic"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// Spec represents a package specification
type Spec struct {
	// version is the version requested, what is included in the package, possibly without prerelease tags.
	version semver.Version
	// specVersion is the version of the spec actually loaded, what can include prerelease tags.
	specVersion semver.Version
	// fs contains the filesystem of the spec.
	fs fs.FS
}

type validationRule func(pkg fspath.FS) specerrors.ValidationErrors

type validationRules []validationRule

// GASpecCheckVersion represents minimum version to start checking for unreleased version of the spec
var GASpecCheckVersion = semver.MustParse("3.0.1")

// NewSpec creates a new Spec for the given version
func NewSpec(version semver.Version) (*Spec, error) {
	specVersion, err := spec.CheckVersion(version)
	if err != nil {
		return nil, fmt.Errorf("could not load specification for version [%s]: %w", version.String(), err)
	}

	// With more current versions this is reported as a filterable validation error for GA packages.
	if version.LessThan(GASpecCheckVersion) {
		if specVersion.Prerelease() != "" {
			log.Printf("Warning: package using an unreleased version of the spec (%s)", specVersion)
		}
	}

	s := Spec{
		version,
		*specVersion,
		spec.FS(),
	}

	return &s, nil
}

// ValidatePackage validates the given Package against the Spec
func (s Spec) ValidatePackage(pkg packages.Package) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	rootSpec, err := loader.LoadSpec(s.fs, s.version, pkg.Type)
	if err != nil {
		errs = append(errs, specerrors.NewStructuredErrorf("could not read root folder spec file: %w", err))
		return errs
	}

	if !s.version.LessThan(GASpecCheckVersion) && pkg.IsGA() {
		if s.specVersion.Prerelease() != "" {
			err := specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\": package with GA version (%s) is using an unreleased version of the spec (%s)", pkg.Path("manifest.yml"), pkg.Version, s.specVersion),
				specerrors.CodeNonGASpecOnGAPackage)
			errs = append(errs, err)
		}
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

func processErrors(errs specerrors.ValidationErrors) specerrors.ValidationErrors {
	var processedErrs specerrors.ValidationErrors

	// Rename unclear error messages
	msgTransforms := []struct {
		matcher *regexp.Regexp
		new     string
	}{
		{
			matcher: regexp.MustCompile(`Must not validate the schema \(not\)`),
			new:     "Must not be present",
		},
		{
			matcher: regexp.MustCompile("secret is required"),
			new:     "variable identified as possible secret, secret parameter required to be set to true or false",
		},
		{
			matcher: regexp.MustCompile(`(field processors.[0-9]+.rename): if is required`),
			new:     "%s: rename \"message\" to \"event.original\" processor requires if: 'ctx.event?.original == null'",
		},
		{
			matcher: regexp.MustCompile(`(field processors.[0-9]+): remove is required`),
			new:     "%s: rename \"message\" to \"event.original\" processor requires remove \"message\" processor",
		},
		{
			matcher: regexp.MustCompile(`(processors.[0-9]+.remove.field(.[0-9]+)?): processors.[0-9]+.remove.field(.[0-9]+)? does not match: "message"`),
			new:     "%s: rename \"message\" to \"event.original\" processor requires remove \"message\" processor",
		},
		{
			matcher: regexp.MustCompile(`(processors.[0-9]+.remove.if): processors.[0-9]+.remove.if does not match: "ctx\.event\?\.original != null"`),
			new:     "%s: rename \"message\" to \"event.original\" processor requires remove \"message\" processor with if: 'ctx.event?.original != null'",
		},
		{
			matcher: regexp.MustCompile(`(field processors.[0-9]+.remove): (ignore_missing|if) is required`),
			new:     "%s: rename \"message\" to \"event.original\" processor requires remove \"message\" processor with if: 'ctx.event?.original != null'",
		},
	}

	// Filter out redundant errors
	redundant := []string{
		"Must validate \"then\" as \"if\" was valid",
		"Must validate \"else\" as \"if\" was not valid",
		"Must validate all the schemas (allOf)",
		"Must validate at least one schema (anyOf)",
		"Must validate one and only one schema (oneOf)",
		"At least one of the items must match",
	}

	// Add error code to specific errors
	addErrorCode := []struct {
		matcher *regexp.Regexp
		code    string
	}{
		{
			matcher: regexp.MustCompile(`rename "message" to "event.original" processor`),
			code:    specerrors.MessageRenameToEventOriginalValidation,
		},
	}

	for _, e := range errs {
		for _, msg := range msgTransforms {
			if match := msg.matcher.FindStringSubmatch(e.Error()); len(match) > 1 {
				e = specerrors.NewStructuredError(
					errors.New(strings.Replace(e.Error(), match[0], fmt.Sprintf(msg.new, match[1]), 1)),
					specerrors.UnassignedCode)
			} else if msg.matcher.MatchString(e.Error()) {
				e = specerrors.NewStructuredError(
					errors.New(strings.Replace(e.Error(), msg.matcher.FindString(e.Error()), msg.new, 1)),
					specerrors.UnassignedCode)
			}
		}
		for _, transform := range addErrorCode {
			if transform.matcher.MatchString(e.Error()) {
				e = specerrors.NewStructuredError(e, transform.code)
			}
		}
		if substringInSlice(e.Error(), redundant) {
			continue
		}
		processedErrs = append(processedErrs, e)
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
		{fn: semantic.WarnOn(semantic.ValidateMinimumKibanaVersion), until: semver.MustParse("3.0.0")},
		{fn: semantic.ValidateMinimumKibanaVersion, since: semver.MustParse("3.0.0")},
		{fn: semantic.ValidateFieldGroups},
		{fn: semantic.ValidateFieldsLimits(rootSpec.MaxFieldsPerDataStream()), types: []string{"integration", "input"}},
		{fn: semantic.ValidateUniqueFields, since: semver.MustParse("2.0.0"), types: []string{"integration", "input"}},
		{fn: semantic.ValidateDimensionFields, types: []string{"integration", "input"}},
		{fn: semantic.ValidateDateFields, types: []string{"integration", "input"}},
		{fn: semantic.ValidateRequiredFields, types: []string{"integration", "input"}},
		{fn: semantic.ValidateExternalFieldsWithDevFolder, types: []string{"integration", "input"}},
		{fn: semantic.WarnOn(semantic.ValidateVisualizationsUsedByValue), types: []string{"integration", "content"}, until: semver.MustParse("3.0.0")},
		{fn: semantic.ValidateVisualizationsUsedByValue, types: []string{"integration", "content"}, since: semver.MustParse("3.0.0")},
		{fn: semantic.ValidateILMPolicyPresent, since: semver.MustParse("2.0.0"), types: []string{"integration"}},
		{fn: semantic.ValidateProfilingNonGA, types: []string{"integration"}},
		{fn: semantic.ValidateKibanaObjectIDs, types: []string{"integration", "content"}},
		{fn: semantic.ValidateRoutingRulesAndDataset, types: []string{"integration"}, since: semver.MustParse("2.9.0")},
		{fn: semantic.ValidateKibanaNoDanglingObjectIDs, since: semver.MustParse("3.0.0")},
		{fn: semantic.ValidateKibanaFilterPresent, since: semver.MustParse("3.0.0")},
		{fn: semantic.ValidateKibanaNoLegacyVisualizations, types: []string{"integration", "content"}, since: semver.MustParse("3.0.0")},
		{fn: semantic.ValidateDimensionsPresent, types: []string{"integration"}, since: semver.MustParse("3.0.1")},
		{fn: semantic.ValidateCapabilitiesRequired, since: semver.MustParse("2.10.0")}, // capabilities definition was added in spec version 2.10.0
		{fn: semantic.ValidateRequiredVarGroups},
		{fn: semantic.ValidateDocsStructure},
		{fn: semantic.ValidateDeploymentModes, types: []string{"integration"}},
		{fn: semantic.ValidateDurationVariables, since: semver.MustParse("3.5.0")},
		{fn: semantic.ValidateInputPackagesPolicyTemplates, types: []string{"input"}},
		{fn: semantic.ValidateMinimumAgentVersion},
		{fn: semantic.ValidateIntegrationPolicyTemplates, types: []string{"integration"}},
		{fn: semantic.ValidatePipelineTags, types: []string{"integration"}, since: semver.MustParse("3.6.0")},
	}

	var validationRules validationRules
	for _, rule := range rulesDef {
		if rule.since != nil && s.version.LessThan(rule.since) {
			continue
		}
		if rule.until != nil && !s.version.LessThan(rule.until) {
			continue
		}

		if rule.types != nil && !slices.Contains(rule.types, pkgType) {
			continue
		}

		validationRules = append(validationRules, rule.fn)
	}

	return validationRules
}

func (vr validationRules) validate(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors
	for _, validationRule := range vr {
		err := validationRule(fsys)
		errs.Append(err)
	}

	return errs
}
