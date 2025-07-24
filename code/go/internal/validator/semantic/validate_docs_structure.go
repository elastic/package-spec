// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	spec "github.com/elastic/package-spec/v3"
	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// EnforcedSections represents the enforced documentation structure configuration.
type EnforcedSections struct {
	Sections []Section `yaml:"enforced_sections"`
}

// Section represents a section in the enforced documentation structure.
type Section struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ValidateDocsStructure validates the structure of documentation files against enforced sections.
func ValidateDocsStructure(fsys fspath.FS) specerrors.ValidationErrors {
	config, err := shouldValidateDocsStructure(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredError(
				fmt.Errorf("failed to determine if documentation structure validation should be performed: %w", err),
				specerrors.UnassignedCode,
			),
		}
	}
	if config == nil {
		return nil
	}

	enforcedSections, err := readDocsStructureEnforcedConfig(fsys, config)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredError(
				fmt.Errorf("failed to read enforced documentation structure configuration: %w", err),
				specerrors.UnassignedCode,
			),
		}
	}

	err = validateReadmeStructure(fsys, enforcedSections)

	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredError(
				err,
				specerrors.UnassignedCode,
			),
		}
	}
	return nil
}

func shouldValidateDocsStructure(fsys fspath.FS) (*specerrors.DocsStructureEnforced, error) {
	validationPath := "validation.yml"
	files, err := pkgpath.Files(fsys, validationPath)
	if err != nil || len(files) == 0 {
		return nil, nil
	}

	validationFile := files[0]
	data, err := validationFile.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", validationFile.Path(), err)
	}

	var config specerrors.ConfigFilter
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML from %s: %w", validationFile.Path(), err)
	}

	if config.DocsStructureEnforced.Enabled {
		return &config.DocsStructureEnforced, nil
	}

	return nil, nil
}

func readDocsStructureEnforcedConfig(fsys fspath.FS, config *specerrors.DocsStructureEnforced) ([]string, error) {
	defaultSections, err := loadSectionsFromConfig(fmt.Sprintf("%d", config.Version))
	if err != nil {
		return nil, fmt.Errorf("failed to load enforced sections from config: %w", err)
	}

	var sections []string
	for _, section := range defaultSections {
		if contains(config.DocsStructureSkip, section) {
			continue
		}
		sections = append(sections, section)
	}

	return sections, nil
}

func contains(slice []specerrors.DocsStructureSkip, item string) bool {
	for _, s := range slice {
		if s.Title == item {
			return true
		}
	}
	return false
}

func loadSectionsFromConfig(version string) ([]string, error) {
	schemaPath := fmt.Sprintf("enforced_sections_v%s.yml", version)

	data, err := fs.ReadFile(spec.DocsFS(), schemaPath)

	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var spec EnforcedSections
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("invalid schema YAML: %w", err)
	}

	sections := make([]string, 0, len(spec.Sections))
	for _, section := range spec.Sections {
		if section.Name != "" {
			sections = append(sections, section.Name)
		}
	}

	return sections, nil
}

func validateReadmeStructure(fsys fspath.FS, enforcedSections []string) error {
	var errs []error
	files, err := pkgpath.Files(fsys, "docs/*.md")
	if err != nil {
		return fmt.Errorf("docs folder %s not found: %w", "docs/*.md", err)
	}
	if len(files) == 0 {
		return nil
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fullPath := path.Join("docs", file.Name())
		content, err := file.ReadAll()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to read file %s: %w", fullPath, err))
		}

		validationErr := validateDocsStructureContent(content, enforcedSections, file.Name())
		if validationErr != nil {
			errs = append(errs, validationErr)
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}

// validateDocsStructureContent validates the content of a documentation file against enforced sections.
func validateDocsStructureContent(content []byte, enforcedSections []string, filename string) error {
	var errs []error
	md := goldmark.New()
	reader := text.NewReader(content)
	doc := md.Parser().Parse(reader)

	found := map[string]bool{}
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if heading, ok := n.(*ast.Heading); ok && entering {
			text := extractHeadingText(heading, content)
			for _, required := range enforcedSections {
				if strings.EqualFold(text, required) {
					found[required] = true
				}
			}
		}
		return ast.WalkContinue, nil
	})

	for _, header := range enforcedSections {
		if !found[header] {
			errs = append(errs, fmt.Errorf("missing required section '%s' in file '%s'", header, filename))
		}
	}

	return errors.Join(errs...)
}

// extractHeadingText extracts the text content from a heading node in the AST.
func extractHeadingText(n ast.Node, source []byte) string {
	var builder strings.Builder

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch t := child.(type) {
		case *ast.Text:
			builder.Write(t.Segment.Value(source))
		}
	}

	return builder.String()
}
