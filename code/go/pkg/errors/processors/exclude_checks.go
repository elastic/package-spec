// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processors

import (
	"github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// Exclude is a processor to filter errors according to their messages.
type ExcludeCheck struct {
	code string
}

// NewExclude creates a new Exclude processor.
func NewExcludeCheck(code string) *ExcludeCheck {
	return &ExcludeCheck{
		code: code,
	}
}

// Name returns the name of this Exclude processor.
func (p ExcludeCheck) Name() string {
	return "exclude-checks"
}

// Process returns a new list of validation errors filtered.
func (p ExcludeCheck) Process(issues errors.ValidationErrors) (errors.ValidationErrors, errors.ValidationErrors, error) {
	if p.code == "" {
		return issues, nil, nil
	}

	errs, filtered := issues.Filter(func(i errors.ValidationError) bool {
		return p.code != i.Code()
	})
	return errs, filtered, nil
}
