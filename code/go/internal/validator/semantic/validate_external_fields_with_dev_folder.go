// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/pkgpath"
	pve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// ValidateExternalFieldsWithDevFolder verifies there is no field with external key if there is no _dev/build/build.yml definition
func ValidateExternalFieldsWithDevFolder(fsys fspath.FS) pve.ValidationErrors {

	const buildPath = "_dev/build/build.yml"
	buildFilePathDefined := true
	f, err := pkgpath.Files(fsys, buildPath)
	if err != nil {
		vError := ve.NewStructuredError(
			fmt.Errorf("not able to read _dev/build/build.yml: %w", err),
			buildPath,
			"",
			ve.Critical,
		)
		return pve.ValidationErrors{vError}
	}

	if len(f) != 1 {
		buildFilePathDefined = false
	}

	mapDependencies := make(map[string]struct{})
	if buildFilePathDefined {
		dependencies, err := readDevBuildDependenciesKeys(f[0])
		if err != nil {
			// TODO
			vError := ve.NewStructuredError(err, buildPath, "", ve.Critical)
			return pve.ValidationErrors{vError}
		}
		for _, dep := range dependencies {
			mapDependencies[dep] = struct{}{}
		}
	}

	validateFunc := func(metadata fieldFileMetadata, f field) pve.ValidationErrors {
		if f.External == "" {
			return nil
		}

		if !buildFilePathDefined {
			vError := ve.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: field %s with external key defined (%q) but no _dev/build/build.yml found", metadata.fullFilePath, f.Name, f.External),
				metadata.filePath,
				"",
				ve.Critical,
			)
			return pve.ValidationErrors{vError}
		}

		if _, ok := mapDependencies[f.External]; !ok {
			// TODO
			vError := ve.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: field %s with external key defined (%q) but no definition found for it (_dev/build/build.yml)", metadata.fullFilePath, f.Name, f.External),
				metadata.filePath,
				"",
				ve.Critical,
			)
			return pve.ValidationErrors{vError}
		}
		return nil
	}
	return validateFields(fsys, validateFunc)
}

func readDevBuildDependenciesKeys(f pkgpath.File) ([]string, error) {
	vals, err := f.Values("$.dependencies")
	if err != nil {
		return []string{}, fmt.Errorf("can't read dependencies: %w", err)
	}

	dependencies, ok := vals.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("dependencies expected to be a map, found %T: %s", vals, vals)
	}

	var keys []string
	for k := range dependencies {
		keys = append(keys, k)
	}

	return keys, nil
}
