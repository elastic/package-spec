// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"io/fs"
	"os"
	"path"

	"github.com/aymerick/raymond"
	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/linkedfiles"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

var (
	errInvalidHandlebarsTemplate = errors.New("invalid handlebars template")
)

// ValidateStaticHandlebarsFiles validates all Handlebars (.hbs) files in the package filesystem.
// It returns a list of validation errors if any Handlebars files are invalid.
// hbs are located in both the package root and data stream directories under the agent folder.
func ValidateStaticHandlebarsFiles(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	// template files are placed at /agent/input directory or
	// at the datastream /agent/stream directory
	inputDir := path.Join("agent", "input")
	if inputErrs := validateTemplateDir(fsys, inputDir); inputErrs != nil {
		errs = append(errs, inputErrs...)
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
		streamDir := path.Join("data_stream", dsEntry.Name(), "agent", "stream")
		dsErrs := validateTemplateDir(fsys, streamDir)
		if dsErrs != nil {
			errs = append(errs, dsErrs...)
		}
	}

	return errs
}

// validateTemplateDir validates all Handlebars files in the given directory.
func validateTemplateDir(fsys fspath.FS, dir string) specerrors.ValidationErrors {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("error trying to read :%s", dir),
		}
	}
	var errs specerrors.ValidationErrors
	for _, entry := range entries {
		if path.Ext(entry.Name()) == ".hbs" {
			err := validateStaticHandlebarsEntry(fsys, dir, entry.Name())
			if err != nil {
				errs = append(errs, specerrors.NewStructuredErrorf("%w: error validating %s: %w", errInvalidHandlebarsTemplate, path.Join(dir, entry.Name()), err))
			}
			continue
		}
		if path.Ext(entry.Name()) == ".link" {
			linkFilePath := path.Join(dir, entry.Name())
			linkFile, err := linkedfiles.NewLinkedFile(fsys.Path(linkFilePath))
			if err != nil {
				errs = append(errs, specerrors.NewStructuredErrorf("error reading linked file %s: %w", linkFilePath, err))
				continue
			}
			err = validateStaticHandlebarsEntry(fsys, dir, linkFile.IncludedFilePath)
			if err != nil {
				errs = append(errs, specerrors.NewStructuredErrorf("%w: error validating %s: %w", errInvalidHandlebarsTemplate, path.Join(dir, linkFile.IncludedFilePath), err))
			}
		}
	}
	return errs
}

// validateStaticHandlebarsEntry validates a single Handlebars file located at filePath.
// it parses the file using the raymond library to check for syntax errors.
func validateStaticHandlebarsEntry(fsys fspath.FS, dir, entryName string) error {
	if entryName == "" {
		return nil
	}

	var content []byte
	var err error

	// First try to read from filesystem (works for regular files and files within zip)
	filePath := path.Join(dir, entryName)
	if content, err = fs.ReadFile(fsys, filePath); err != nil {
		// If fs.ReadFile fails (likely due to linked file path outside filesystem boundary),
		// fall back to absolute path approach like linkedfiles.FS does
		absolutePath := fsys.Path(filePath)
		if content, err = os.ReadFile(absolutePath); err != nil {
			return err
		}
	}

	// Parse from content string instead of file path
	_, err = raymond.Parse(string(content))
	return err
}
