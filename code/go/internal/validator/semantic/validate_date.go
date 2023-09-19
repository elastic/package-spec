// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	pve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// ValidateDateFields verifies if date fields are of one of the expected types.
func ValidateDateFields(fsys fspath.FS) pve.ValidationErrors {
	return validateFields(fsys, validateDateField)
}

func validateDateField(metadata fieldFileMetadata, f field) pve.ValidationErrors {
	if f.Type != "date" && f.DateFormat != "" {
		vError := ve.NewStructuredError(
			fmt.Errorf(`file "%s" is invalid: field "%s" of type %s can't set date_format. date_format is allowed for date field type only`, metadata.fullFilePath, f.Name, f.Type),
			metadata.filePath,
			"",
			pve.Critical,
		)
		return pve.ValidationErrors{vError}
	}

	return nil
}
