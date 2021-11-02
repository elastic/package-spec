// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	"github.com/elastic/package-spec/code/go/internal/errors"
)

// ValidateFieldGroups verifies if field groups don't have units and metric types defined.
func ValidateFieldGroups(pkgRoot string) errors.ValidationErrors {
	return validateFields(pkgRoot, validateFieldUnit)
}

func validateFieldUnit(fieldsFile string, f field) errors.ValidationErrors {
	if f.Type == "group" && f.Unit != "" {
		return errors.ValidationErrors{fmt.Errorf(`file "%s" is invalid: field "%s" can't have unit property'`, fieldsFile, f.Name)}
	}

	if f.Type == "group" && f.MetricType != "" {
		return errors.ValidationErrors{fmt.Errorf(`file "%s" is invalid: field "%s" can't have metric type property'`, fieldsFile, f.Name)}
	}

	var vErrs errors.ValidationErrors
	for _, aField := range f.Fields {
		errs := validateFieldUnit(fieldsFile, aField)
		if len(errs) > 0 {
			vErrs = append(vErrs, errs...)
		}
	}
	return vErrs
}
