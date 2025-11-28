// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"io/fs"
	"strings"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
	"github.com/mailgun/raymond/v2"
)

var (
	ErrInvalidHandlebarsTemplate = errors.New("invalid handlebars template")
)

func ValidateHandlebarsFiles(fsys fspath.FS) specerrors.ValidationErrors {
	entries, err := getHandlebarsFiles(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf(
				"error finding Handlebars files: %w", err,
			),
		}
	}
	if len(entries) == 0 {
		return nil
	}

	var validationErrors specerrors.ValidationErrors
	for _, entry := range entries {
		if !strings.HasSuffix(entry, ".hbs") {
			continue
		}

		filePath := fsys.Path(entry)
		err := validateFile(filePath)
		if err != nil {
			validationErrors = append(validationErrors, specerrors.NewStructuredErrorf(
				"%w: file %s: %w", ErrInvalidHandlebarsTemplate, entry, err,
			))
		}
	}

	return validationErrors
}

// validateFile validates a single Handlebars file located at filePath.
// it parses the file using the raymond library to check for syntax errors.
func validateFile(filePath string) error {
	if filePath == "" {
		return nil
	}
	_, err := raymond.ParseFile(filePath)
	return err
}

// getHandlebarsFiles returns all Handlebars (.hbs) files in the package filesystem.
// It searches in both the package root and data stream directories under the agent folder.
func getHandlebarsFiles(fsys fspath.FS) ([]string, error) {
	entries := make([]string, 0)
	pkgEntries, err := fs.Glob(fsys, "agent/**/*.hbs")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	entries = append(entries, pkgEntries...)

	dataStreamEntries, err := fs.Glob(fsys, "data_stream/*/agent/**/*.hbs")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	entries = append(entries, dataStreamEntries...)

	return entries, nil
}
