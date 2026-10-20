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

const (
	esqlViewFolder = "elasticsearch/esql_view"

	// esqlViewNamePrefix is the namespace in Elasticsearch where Fleet is allowed
	// to manage views; the full view name is "<prefix><package>.<view>".
	esqlViewNamePrefix = "$.integrations."
)

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

	packageName, err := readPackageName(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("%v", err)}
	}

	var errs specerrors.ValidationErrors

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

		if strings.TrimSpace(view.Query) == "" {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: query must not be empty or whitespace-only", fullPath),
				specerrors.CodeEsqlViewValidation,
			))
		}

		stem := strings.TrimSuffix(entry.Name(), ".yml")
		expectedName := esqlViewNamePrefix + packageName + "." + stem

		// Enforcing the name to be derived from the filename stem also prevents
		// duplicate view names: filesystem filenames are unique, so two files
		// cannot share the same stem. The package name in the prefix scopes the
		// view to the package so it cannot collide with views from other packages.
		if view.Name != expectedName {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: name field %q must be %q (\"%s\" + package name %q + \".\" + filename stem %q)", fullPath, view.Name, expectedName, esqlViewNamePrefix, packageName, stem),
				specerrors.CodeEsqlViewValidation,
			))
		}
	}

	return errs
}
