// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"os"
	"path"

	"github.com/aymerick/raymond"

	"github.com/elastic/package-spec/v3/code/go/internal/linkedfiles"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

var (
	errInvalidHandlebarsTemplate = errors.New("invalid handlebars template")
)

// ValidateStaticHandlebarsFiles validates all Handlebars (.hbs) files in the package filesystem.
// It returns a list of validation errors if any Handlebars files are invalid.
// hbs are located in both the package root and data stream directories under the agent folder.
func ValidateStaticHandlebarsFiles(fsys PackageFS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	// template files are placed at /agent/input directory or
	// at the datastream /agent/stream directory
	inputDir := path.Join("agent", "input")
	if inputErrs := validateTemplateDir(fsys, inputDir); inputErrs != nil {
		errs = append(errs, inputErrs...)
	}

	datastreamEntries, err := fsys.Files("data_stream/*")
	if err != nil {
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
func validateTemplateDir(fsys PackageFS, dir string) specerrors.ValidationErrors {
	entries, err := fsys.Files(path.Join(dir, "*"))
	if err != nil {
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
func validateStaticHandlebarsEntry(fsys PackageFS, dir, entryName string) error {
	if entryName == "" {
		return nil
	}

	var content []byte
	var err error

	filePath := path.Join(dir, entryName)
	files, err := fsys.Files(filePath)
	if err != nil {
		return err
	}
	if len(files) > 0 {
		content, err = files[0].ReadAll()
		if err != nil {
			return err
		}
	} else {
		// File not found in virtual filesystem, fall back to absolute path
		// for linked files pointing outside the filesystem boundary.
		absolutePath := fsys.Path(filePath)
		if content, err = os.ReadFile(absolutePath); err != nil {
			return err
		}
	}

	// Parse from content string instead of file path
	_, err = raymond.Parse(string(content))
	return err
}
