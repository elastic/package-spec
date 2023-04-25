// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	"github.com/pkg/errors"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/pkgpath"
)

// validateExternalFieldsWithBuildFolder verifies there is no field with external key if there is no _dev/build folder
func ValidateExternalFieldsWithDevFolder(fsys fspath.FS) ve.ValidationErrors {
	buildFilePathDefined := true
	// it has _dev/build/build ?
	if _, err := readECSReference(fsys); err != nil {
		buildFilePathDefined = false
	}

	validateFunc := func(fieldsFile string, f field) ve.ValidationErrors {
		if buildFilePathDefined {
			return nil
		}

		if f.External != "" {
			return ve.ValidationErrors{fmt.Errorf("file \"%s\" is invalid: field %s with external key defined (%q) but no ECS reference found (_dev/build/build.yml)", fieldsFile, f.Name, f.External)}
		}
		return nil
	}
	return validateFields(fsys, validateFunc)
}

func readECSReference(fsys fspath.FS) (string, error) {
	buildPath := "_dev/build/build.yml"
	f, err := pkgpath.Files(fsys, buildPath)
	if err != nil {
		return "", errors.Wrap(err, "can't locate _dev/build/build.yml file")
	}

	if len(f) != 1 {
		return "", errors.New("single build file expected")
	}

	val, err := f[0].Values("$.dependencies.ecs.reference")
	if err != nil {
		return "", errors.Wrap(err, "can't read ecs.reference")
	}

	sVal, ok := val.(string)
	if !ok {
		return "", errors.New("ecs.reference is undefined")
	}
	return sVal, nil
}
