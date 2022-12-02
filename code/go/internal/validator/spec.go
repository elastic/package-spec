// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"io/fs"
	"log"
	"path"
	"path/filepath"
	"regexp"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	spec "github.com/elastic/package-spec/v2"
	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/loader"
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
func (s Spec) ValidatePackage(pkg Package) ve.ValidationErrors {
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
	errs = append(errs, s.rules(rootSpec).validate(&pkg)...)

	return errs
}

func (s Spec) rules(rootSpec spectypes.ItemSpec) validationRules {
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
		{fn: semantic.ValidateFieldsLimits(rootSpec.MaxFieldsPerDataStream())},
		{fn: semantic.ValidateUniqueFields, since: semver.MustParse("2.0.0")},
		{fn: semantic.ValidateDimensionFields},
		{fn: semantic.ValidateRequiredFields},
		{fn: semantic.ValidateVisualizationsUsedByValue},
		{fn: semantic.ValidateILMPolicyPresent, since: semver.MustParse("2.0.0")},
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

func (s Spec) AllJSONSchema(pkgType string) ([]RenderedJSONSchema, error) {
	rootSpec, err := loader.LoadSpec(s.fs, s.version, pkgType)
	if err != nil {
		return nil, err
	}

	contents, err := marshalSpec(rootSpec)
	if err != nil {
		return nil, err
	}

	return contents, nil
}

func (s Spec) JSONSchema(location, pkgType string) (*RenderedJSONSchema, error) {
	var rendered RenderedJSONSchema
	rootSpec, err := loader.LoadSpec(s.fs, s.version, pkgType)
	if err != nil {
		return nil, err
	}

	contents, err := marshalSpec(rootSpec)
	if err != nil {
		return nil, err
	}

	for _, content := range contents {
		matched, err := matchContentWithFile(location, content.Name)
		if err != nil {
			return nil, err
		}
		if !matched {
			continue
		}

		if content.JSONSchema == nil {
			content.JSONSchema = []byte("")
		}

		if len(rendered.JSONSchema) > 0 && len(content.JSONSchema) == 0 {
			// not overwrite rendered with contents that are empty strings (e.g. docs/README.md)
			continue
		}
		rendered = content
		log.Printf("Matched item spec %s for path %s", content.Name, location)
	}
	if rendered.JSONSchema == nil {
		return nil, errors.Errorf("item path not found: %s", location)
	}
	return &rendered, nil
}

type RenderedJSONSchema struct {
	Name       string
	JSONSchema []byte
}

func marshalSpec(spec spectypes.ItemSpec) ([]RenderedJSONSchema, error) {
	var allContents []RenderedJSONSchema
	if len(spec.Contents()) == 0 {
		key := spec.Name()
		if key == "" {
			key = spec.Pattern()
		}
		contents, err := spec.Marshal()
		if err != nil {
			return nil, err
		}

		allContents = append(allContents, RenderedJSONSchema{key, contents})
		return allContents, nil
	}
	pending := spec.Contents()
	for _, item := range pending {
		itemsJSON, err := marshalSpec(item)
		if err != nil {
			return nil, err
		}
		if item.IsDir() {
			for c, elem := range itemsJSON {
				itemsJSON[c].Name = path.Join(item.Name(), elem.Name)
			}
		}
		allContents = append(allContents, itemsJSON...)
	}
	return allContents, nil
}

func matchContentWithFile(location, content string) (bool, error) {
	baseLocation := filepath.Base(location)
	baseContent := filepath.Base(content)

	dirLocation := filepath.Dir(location)
	dirContent := filepath.Dir(content)

	if dirLocation != dirContent {
		return false, nil
	}

	r, err := regexp.Compile(baseContent)
	if err != nil {
		log.Printf(" -- Error %+s", err)
		return false, errors.Wrap(err, "failed to compile regex")
	}
	if !r.MatchString(baseLocation) {
		return false, nil
	}

	return true, nil
}
