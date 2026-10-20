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

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const esqlViewFolder = "elasticsearch/esql_view"

type esqlView struct {
	Name string `yaml:"name"`
	// Query is a pointer so a missing or null query can be told apart from an empty one:
	// the former is already reported by the JSON schema validation.
	Query *string `yaml:"query"`
}

// ValidateEsqlViews checks semantic rules for ES|QL view definitions: view names must be
// unique within a package and queries must not be empty or whitespace-only.
func ValidateEsqlViews(fsys fspath.FS) specerrors.ValidationErrors {
	entries, err := fs.ReadDir(fsys, esqlViewFolder)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("error reading %s: %v", esqlViewFolder, err)}
	}

	var errs specerrors.ValidationErrors
	seen := make(map[string]string) // view name -> full path of the first file defining it

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".yml" {
			continue
		}

		filePath := path.Join(esqlViewFolder, entry.Name())
		fullPath := fsys.Path(filePath)

		b, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("error reading file %s: %v", fullPath, err))
			continue
		}

		var view esqlView
		if err := yaml.Unmarshal(b, &view); err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("error unmarshaling file %s: %v", fullPath, err))
			continue
		}

		if view.Query != nil && strings.TrimSpace(*view.Query) == "" {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: query must not be empty or whitespace-only", fullPath),
				specerrors.CodeEsqlViewValidation,
			))
		}

		if view.Name == "" {
			continue
		}

		if first, found := seen[view.Name]; found {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: ES|QL view name \"%s\" is already defined in file \"%s\"", fullPath, view.Name, first),
				specerrors.CodeEsqlViewValidation,
			))
			continue
		}
		seen[view.Name] = fullPath
	}

	return errs
}
