// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

type anyOfCondition struct {
	path          string
	anyOfPatterns []string
}

func (a *anyOfCondition) validate(fsys fspath.FS) specerrors.ValidationErrors {
	if len(a.anyOfPatterns) == 0 || a.path == "" {
		return nil
	}

	var errs specerrors.ValidationErrors
	if err := a.validatePath(fsys, a.path); err != nil {
		errs = append(errs, specerrors.NewStructuredErrorf("path %q: %w", a.path, err))
	}

	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	for _, dataStream := range dataStreams {
		path := filepath.Join("data_stream", dataStream, a.path)
		err := a.validatePath(fsys, path)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("data stream %q: %w", dataStream, err))
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (a *anyOfCondition) validatePath(fsys fspath.FS, path string) specerrors.ValidationError {
	files, err := fs.ReadDir(fsys, path)
	if err != nil {
		if !os.IsNotExist(err) {
			return specerrors.NewStructuredError(err, specerrors.UnassignedCode)
		}
		return nil
	}
	for _, file := range files {
		for _, pattern := range a.anyOfPatterns {
			matched, err := filepath.Match(pattern, file.Name())
			if err != nil {
				return specerrors.NewStructuredError(err, specerrors.UnassignedCode)
			}
			if matched {
				return nil
			}
		}
	}
	return specerrors.NewStructuredErrorf("no file matching any of the patterns %v found in %s", a.anyOfPatterns, path)
}

// ValidateAnyOfRequiredContents validates that at least one file matching
// any of the patterns in the given path exists in the package.
// It checks the following paths:
// - agent/input
// - agent/stream
// - fields
// - elasticsearch/ingest_pipeline
func ValidateAnyOfRequiredContents(fsys fspath.FS) specerrors.ValidationErrors {
	conditions := []anyOfCondition{
		{path: filepath.Join("agent", "input"), anyOfPatterns: []string{"*.yml.hbs", "*.yml.hbs.link"}},
		{path: filepath.Join("agent", "stream"), anyOfPatterns: []string{"*.yml.hbs", "*.yml.hbs.link"}},
		{path: "fields", anyOfPatterns: []string{"*.yml", "*.yml.link"}},
		{path: filepath.Join("elasticsearch", "ingest_pipeline"), anyOfPatterns: []string{"*.yml", "*.json", "*.yml.link", "*.json.link"}},
	}

	var errs specerrors.ValidationErrors
	for _, c := range conditions {
		cerrs := c.validate(fsys)
		if cerrs != nil {
			errs = append(errs, cerrs...)
		}
	}
	return errs
}
