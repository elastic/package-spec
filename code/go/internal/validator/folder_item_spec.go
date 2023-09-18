// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"fmt"
	"io/fs"
	"regexp"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/spectypes"
)

func matchingFileExists(spec spectypes.ItemSpec, files []fs.DirEntry) (bool, error) {
	if spec.Name() != "" {
		for _, file := range files {
			if file.Name() == spec.Name() {
				return spec.IsDir() == file.IsDir(), nil
			}
		}
	} else if spec.Pattern() != "" {
		for _, file := range files {
			isMatch, err := regexp.MatchString(spec.Pattern(), file.Name())
			if err != nil {
				return false, fmt.Errorf("invalid folder item spec pattern: %w", err)
			}
			if isMatch {
				return spec.IsDir() == file.IsDir(), nil
			}
		}
	}

	return false, nil
}

func validateFile(spec spectypes.ItemSpec, fsys fs.FS, itemPath string) ve.ValidationErrors {
	err := validateMaxSize(fsys, itemPath, spec)
	if err != nil {
		vError := ve.NewStructuredError(err, itemPath, "", ve.Critical)
		return ve.ValidationErrors{vError}
	}
	if mediaType := spec.ContentMediaType(); mediaType != nil {
		err := validateContentType(fsys, itemPath, *mediaType)
		if err != nil {
			vError := ve.NewStructuredError(err, itemPath, "", ve.Critical)
			return ve.ValidationErrors{vError}
		}
		err = validateContentTypeSize(fsys, itemPath, *mediaType, spec)
		if err != nil {
			vError := ve.NewStructuredError(err, itemPath, "", ve.Critical)
			return ve.ValidationErrors{vError}
		}
	}

	errs := spec.ValidateSchema(fsys, itemPath)
	if len(errs) > 0 {
		return errs
	}

	return nil
}
