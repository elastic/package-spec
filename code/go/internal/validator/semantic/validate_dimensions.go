// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"strings"

	"github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
)

// ValidateDimensionFields verifies if dimension fields are of one of the expected types.
func ValidateDimensionFields(fsys fspath.FS) errors.ValidationErrors {
	return validateFields(fsys, validateDimensionField)
}

func validateDimensionField(metadata fieldFileMetadata, f field) errors.ValidationErrors {
	if f.External != "" {
		// TODO: External fields can be used as dimensions, but we cannot resolve
		// them at this point, so accept them as they are by now.
		return nil
	}
	if f.Dimension && !isAllowedDimensionType(f.Type) {
		vError := errors.NewStructuredError(
			fmt.Errorf(`file "%s" is invalid: field "%s" of type %s can't be a dimension, allowed types for dimensions: %s`, metadata.fullFilePath, f.Name, f.Type, strings.Join(allowedDimensionTypes, ", ")),
			metadata.filePath,
			"",
			errors.Critical,
		)
		return errors.ValidationErrors{vError}
	}

	return nil
}

var allowedDimensionTypes = []string{
	// Keywords
	"constant_keyword", "keyword",

	// Numeric types
	"long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "unsigned_long",

	// IPs
	"ip",
}

func isAllowedDimensionType(fieldType string) bool {
	for _, allowedType := range allowedDimensionTypes {
		if fieldType == allowedType {
			return true
		}
	}

	return false
}
