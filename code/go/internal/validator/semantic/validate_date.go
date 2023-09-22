// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// ValidateDateFields verifies if date fields are of one of the expected types.
func ValidateDateFields(fsys fspath.FS) errors.ValidationErrors {
	return validateFields(fsys, validateDateField)
}

func validateDateField(metadata fieldFileMetadata, f field) errors.ValidationErrors {
	if f.Type != "date" && f.DateFormat != "" {
		err := fmt.Errorf(`file "%s" is invalid: field "%s" of type %s can't set date_format. date_format is allowed for date field type only`, metadata.fullFilePath, f.Name, f.Type)

		return errors.ValidationErrors{errors.NewStructuredError(err, errors.TODO_code)}
	}

	return nil
}
