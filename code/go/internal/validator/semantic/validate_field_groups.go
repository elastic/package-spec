// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	"github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
)

// ValidateFieldGroups verifies if field groups don't have units and metric types defined.
func ValidateFieldGroups(fsys fspath.FS) errors.ValidationErrors {
	return validateFields(fsys, validateFieldUnit)
}

func validateFieldUnit(metadata fieldFileMetadata, f field) errors.ValidationErrors {
	if f.Type == "group" && f.Unit != "" {
		return errors.ValidationErrors{fmt.Errorf(`file "%s" is invalid: field "%s" can't have unit property'`, metadata.fullFilePath, f.Name)}
	}

	if f.Type == "group" && f.MetricType != "" {
		return errors.ValidationErrors{fmt.Errorf(`file "%s" is invalid: field "%s" can't have metric type property'`, metadata.fullFilePath, f.Name)}
	}

	return nil
}
