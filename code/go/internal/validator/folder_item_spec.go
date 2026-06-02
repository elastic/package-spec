// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"fmt"
	"io/fs"
	"regexp"

	"github.com/elastic/package-spec/v3/code/go/internal/spectypes"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

func matchingFileExists(spec spectypes.ItemSpec, files []fs.DirEntry) (bool, error) {
	if spec.Name() != "" {
		for _, file := range files {
			_, fileName := checkLink(file.Name())
			if fileName == spec.Name() {
				return spec.IsDir() == file.IsDir(), nil
			}
		}
	} else if spec.Pattern() != "" {
		for _, file := range files {
			_, fileName := checkLink(file.Name())
			isMatch, err := regexp.MatchString(spec.Pattern(), fileName)
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

func validateFile(spec spectypes.ItemSpec, fsys fs.FS, itemPath string) specerrors.ValidationErrors {
	err := validateMaxSize(fsys, itemPath, spec)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}
	if mediaType := spec.ContentMediaType(); mediaType != nil {
		err := validateContentType(fsys, itemPath, *mediaType)
		if err != nil {
			return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
		}
		err = validateContentTypeSize(fsys, itemPath, *mediaType, spec)
		if err != nil {
			return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
		}
	}

	errs := spec.ValidateSchema(fsys, itemPath)
	if len(errs) > 0 {
		return errs
	}

	return nil
}
