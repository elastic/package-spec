// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/dlclark/regexp2"
	jsparser "github.com/dop251/goja/parser"

	"github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/pkgpath"
)

// ValidateFormatPatterns verifies if format patterns are valid ECMAScript regular expressions.
func ValidateFormatPatterns(packageRoot string) errors.ValidationErrors {
	manifests, err := findManifests(packageRoot)
	if err != nil {
		return errors.ValidationErrors{err}
	}

	var errs errors.ValidationErrors
	for _, manifest := range manifests {
		patterns, err := findPatterns(manifest)
		if err != nil {
			return errors.ValidationErrors{err}
		}

		for _, pattern := range patterns {
			err := validateFormatPattern(pattern)
			if err != nil {
				errs = append(errs, fmt.Errorf(`file "%s" is invalid: format pattern (%s) is invalid: %v`, manifest.Path(), pattern, err))
			}
		}
	}

	return errs
}

func validateFormatPattern(pattern string) error {
	if len(pattern) == 0 {
		return nil
	}

	// Parsing based on https://github.com/dop251/goja/blob/dc8c55024d06009a0cba080cd738192d368735de/builtin_regexp.go#L235
	transformed, err := jsparser.TransformRegExp(pattern)

	// If no error, transformed pattern is compatible with Go regexp.
	if err == nil {
		_, err := regexp.Compile(transformed)
		return err
	}

	if _, incompatible := err.(jsparser.RegexpErrorIncompatible); !incompatible {
		// Pattern is not valid.
		return err
	}

	// Pattern is valid, but cannot be compiled by Go regexp, try with regexp2.
	var opts regexp2.RegexOptions = regexp2.ECMAScript
	_, err = regexp2.Compile(pattern, opts)
	return err
}

func findManifests(packageRoot string) ([]pkgpath.File, error) {
	manifests, err := pkgpath.Files(filepath.Join(packageRoot, "manifest.yml"))
	if err != nil {
		return nil, fmt.Errorf("can't locate manifest file in %s: %v", packageRoot, err)
	}
	if len(manifests) == 0 {
		return nil, fmt.Errorf("cant't locate manifest file in %s", packageRoot)
	}

	dataStreamManifests, err := pkgpath.Files(filepath.Join(packageRoot, "data_stream/*/manifest.yml"))
	if err != nil {
		return nil, fmt.Errorf("can't locate data stream manifests in %s: %v", packageRoot, err)
	}

	return append(manifests, dataStreamManifests...), nil
}

func findPatterns(manifest pkgpath.File) ([]string, error) {
	jsonPaths := []string{
		// Patterns in package manifest
		"$.policy_templates[*].inputs[*].vars[*].format.pattern",
		// Patterns in datastream manifests
		"$.streams[*].vars[*].format.pattern",
	}

	var patterns []string
	for _, jsonPath := range jsonPaths {
		values, err := manifest.Values(jsonPath)
		if err != nil {
			return nil, err
		}

		switch values := values.(type) {
		case []string:
			patterns = append(patterns, values...)
		case []interface{}:
			for _, v := range values {
				if v, ok := v.(string); ok {
					patterns = append(patterns, v)
				} else {
					return nil, fmt.Errorf("a format pattern is not a string")
				}
			}
		}
	}

	return patterns, nil
}
