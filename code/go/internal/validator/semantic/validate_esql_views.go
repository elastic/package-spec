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
	Name  string `yaml:"name"`
	Query string `yaml:"query"`
}

// ValidateEsqlViews checks semantic rules for ES|QL view definitions.
func ValidateEsqlViews(fsys fspath.FS) specerrors.ValidationErrors {
	entries, err := fs.ReadDir(fsys, esqlViewFolder)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("error reading %s: %v", esqlViewFolder, err)}
	}

	var errs specerrors.ValidationErrors
	seenNames := make(map[string]struct{})

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

		stem := strings.TrimSuffix(entry.Name(), ".yml")

		if view.Name != stem {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: name field %q must equal the filename stem %q", fullPath, view.Name, stem),
				specerrors.CodeEsqlViewValidation,
			))
		}

		if view.Name == "." || view.Name == ".." {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: name %q is not allowed", fullPath, view.Name),
				specerrors.CodeEsqlViewValidation,
			))
		}

		if strings.TrimSpace(view.Query) == "" {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: query must not be empty or whitespace-only", fullPath),
				specerrors.CodeEsqlViewValidation,
			))
		}

		if _, seen := seenNames[view.Name]; seen {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: duplicate view name %q", fullPath, view.Name),
				specerrors.CodeEsqlViewValidation,
			))
		} else if view.Name == stem {
			seenNames[view.Name] = struct{}{}
		}
	}

	return errs
}
