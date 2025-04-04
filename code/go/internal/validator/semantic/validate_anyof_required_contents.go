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

type AnyOfCondition struct {
	Path          string
	AnyOfPatterns []string
}

func (a *AnyOfCondition) Validate(fsys fspath.FS) specerrors.ValidationErrors {
	if len(a.AnyOfPatterns) == 0 || a.Path == "" {
		return nil
	}

	var errs specerrors.ValidationErrors
	if err := a.validatePath(fsys, a.Path); err != nil {
		errs = append(errs, specerrors.NewStructuredErrorf("path %q: %w", a.Path, err))
	}

	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	for _, dataStream := range dataStreams {
		path := filepath.Join("data_stream", dataStream, a.Path)
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

func (a *AnyOfCondition) validatePath(fsys fspath.FS, path string) specerrors.ValidationError {
	files, err := fs.ReadDir(fsys, path)
	if err != nil {
		if !os.IsNotExist(err) {
			return specerrors.NewStructuredError(err, specerrors.UnassignedCode)
		}
		return nil
	}
	for _, file := range files {
		for _, pattern := range a.AnyOfPatterns {
			matched, err := filepath.Match(pattern, file.Name())
			if err != nil {
				return specerrors.NewStructuredError(err, specerrors.UnassignedCode)
			}
			if matched {
				return nil
			}
		}
	}
	return specerrors.NewStructuredErrorf("no file matching any of the patterns %v found in %s", a.AnyOfPatterns, path)
}

func ValidateAnyOfRequiredContents(fsys fspath.FS) specerrors.ValidationErrors {
	conditions := []AnyOfCondition{
		{Path: filepath.Join("agent", "input"), AnyOfPatterns: []string{"*.yml.hbs", "*.yml.hbs.link"}},
		{Path: filepath.Join("agent", "stream"), AnyOfPatterns: []string{"*.yml.hbs", "*.yml.hbs.link"}},
		{Path: "fields", AnyOfPatterns: []string{"*.yml", "*.yml.link"}},
		{Path: filepath.Join("elasticsearch", "ingest_pipeline"), AnyOfPatterns: []string{"*.yml", "*.json", "*.yml.link", "*.json.link"}},
	}

	var errs specerrors.ValidationErrors
	for _, c := range conditions {
		cerrs := c.Validate(fsys)
		if cerrs != nil {
			errs = append(errs, cerrs...)
		}
	}
	return errs
}
