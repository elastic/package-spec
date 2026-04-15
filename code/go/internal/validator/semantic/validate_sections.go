// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateSections validates sections definitions in manifests.
// It checks that:
// - section names are unique within each scope
// - vars that reference a section via the `section` attribute name a section
// defined in the `sections` list at the same scope level
func ValidateSections(fsys fspath.FS) specerrors.ValidationErrors {
	d, err := fs.ReadFile(fsys, "manifest.yml")
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("failed to read file \"%s\": %w", fsys.Path("manifest.yml"), err)}
	}

	var manifest sectionsManifest
	if err := yaml.Unmarshal(d, &manifest); err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path("manifest.yml"), err)}
	}

	errs := validateSectionsManifest(fsys.Path("manifest.yml"), manifest)

	// Validate data stream manifests.
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("failed to list data streams: %w", err)}
	}
	for _, ds := range dataStreams {
		errs = append(errs, validateDataStreamSections(fsys, path.Join("data_stream", ds, "manifest.yml"))...)
	}

	return errs
}

type sectionsVar struct {
	Name    string `yaml:"name"`
	Section string `yaml:"section"`
}

type manifestSection struct {
	Name string `yaml:"name"`
}

type sectionsManifest struct {
	Sections        []manifestSection `yaml:"sections"`
	Vars            []sectionsVar     `yaml:"vars"`
	PolicyTemplates []struct {
		Sections []manifestSection `yaml:"sections"`
		Vars     []sectionsVar     `yaml:"vars"`
		Inputs   []struct {
			Sections []manifestSection `yaml:"sections"`
			Vars     []sectionsVar     `yaml:"vars"`
		} `yaml:"inputs"`
	} `yaml:"policy_templates"`
}

type sectionsDataStreamManifest struct {
	Streams []struct {
		Title    string            `yaml:"title"`
		Input    string            `yaml:"input"`
		Sections []manifestSection `yaml:"sections"`
		Vars     []sectionsVar     `yaml:"vars"`
	} `yaml:"streams"`
}

func validateSectionsManifest(filePath string, manifest sectionsManifest) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	errs = append(errs, validateSectionsScope(filePath, "package root", manifest.Sections, manifest.Vars)...)

	for _, pt := range manifest.PolicyTemplates {
		errs = append(errs, validateSectionsScope(filePath, "policy template", pt.Sections, pt.Vars)...)
		for _, input := range pt.Inputs {
			errs = append(errs, validateSectionsScope(filePath, "input", input.Sections, input.Vars)...)
		}
	}

	return errs
}

func validateDataStreamSections(fsys fspath.FS, filePath string) specerrors.ValidationErrors {
	d, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		// File might not exist, which is fine.
		return nil
	}

	var manifest sectionsDataStreamManifest
	if err := yaml.Unmarshal(d, &manifest); err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(filePath), err)}
	}

	var errs specerrors.ValidationErrors
	for i, stream := range manifest.Streams {
		streamID := stream.Title
		if streamID == "" {
			streamID = stream.Input
		}
		if streamID == "" {
			streamID = fmt.Sprintf("stream[%d]", i)
		}
		errs = append(errs, validateSectionsScope(fsys.Path(filePath), fmt.Sprintf("stream %q", streamID), stream.Sections, stream.Vars)...)
	}

	return errs
}

func validateSectionsScope(filePath, scope string, sections []manifestSection, vars []sectionsVar) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	// Build set of defined section names, checking for duplicates.
	sectionNames := make(map[string]bool)
	for _, s := range sections {
		if sectionNames[s.Name] {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: duplicate section name %q in %s", filePath, s.Name, scope))
		}
		sectionNames[s.Name] = true
	}

	// Verify that each var's section attribute references a defined section.
	for _, v := range vars {
		if v.Section == "" {
			continue
		}
		if !sectionNames[v.Section] {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: var %q references undefined section %q in %s", filePath, v.Name, v.Section, scope))
		}
	}

	return errs
}
