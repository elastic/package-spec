// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/mailgun/raymond/v2"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/linkedfiles"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

var (
	errInvalidHandlebarsTemplate = errors.New("invalid handlebars template")
)

// ValidateHandlebarsFiles validates all Handlebars (.hbs) files in the package filesystem.
// It returns a list of validation errors if any Handlebars files are invalid.
// hbs are located in both the package root and data stream directories under the agent folder.
func ValidateHandlebarsFiles(fsys fspath.FS) specerrors.ValidationErrors {
	// template files are placed at /agent/input directory or
	// at the datastream /agent/stream directory
	inputDir := filepath.Join("agent", "input")
	err := validateTemplateDir(fsys, inputDir)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("%w in %s: %w", errInvalidHandlebarsTemplate, inputDir, err),
		}
	}

	datastreamEntries, err := fs.ReadDir(fsys, "data_stream")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("error reading data_stream directory: %w", err),
		}
	}
	for _, dsEntry := range datastreamEntries {
		if !dsEntry.IsDir() {
			continue
		}
		streamDir := filepath.Join("data_stream", dsEntry.Name(), "agent", "stream")
		err := validateTemplateDir(fsys, streamDir)
		if err != nil {
			return specerrors.ValidationErrors{
				specerrors.NewStructuredErrorf("%w in %s: %w", errInvalidHandlebarsTemplate, streamDir, err),
			}
		}
	}

	return nil
}

// validateTemplateDir validates all Handlebars files in the given directory.
func validateTemplateDir(fsys fspath.FS, dir string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".hbs" {
			err := validateHandlebarsEntry(fsys, dir, entry.Name())
			if err != nil {
				return err
			}
			continue
		}
		if filepath.Ext(entry.Name()) == ".link" {
			linkFilePath := path.Join(dir, entry.Name())
			linkFile, err := linkedfiles.NewLinkedFile(fsys.Path(linkFilePath))
			if err != nil {
				return err
			}
			err = validateHandlebarsEntry(fsys, dir, linkFile.IncludedFilePath)
			if err != nil {
				return err
			}
			continue
		}
	}
	return nil
}

// validateHandlebarsEntry validates a single Handlebars file located at filePath.
// it parses the file using the raymond library to check for syntax errors.
func validateHandlebarsEntry(fsys fspath.FS, dir, entryName string) error {
	if entryName == "" {
		return nil
	}
	filePath := fsys.Path(path.Join(dir, entryName))
	_, err := raymond.ParseFile(filePath)
	return err
}
