// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processors

import (
	"github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// ExcludeCheck is a processor to filter errors according to their messages.
type ExcludeCheck struct {
	code string
}

// NewExcludeCheck creates a new ExcludeCheck processor.
func NewExcludeCheck(code string) *ExcludeCheck {
	return &ExcludeCheck{
		code: code,
	}
}

// Name returns the name of this ExcludeCheck processor.
func (p ExcludeCheck) Name() string {
	return "exclude-checks"
}

// Process returns a new list of validation errors filtered.
func (p ExcludeCheck) Process(issues errors.ValidationErrors) (errors.ValidationErrors, errors.ValidationErrors, error) {
	if p.code == errors.UnassignedCode {
		return issues, nil, nil
	}

	errs, filtered := issues.Collect(func(i errors.ValidationError) bool {
		if i.Code() == errors.UnassignedCode {
			// errors with TODO_code cannot be skipped
			return true
		}
		return p.code != i.Code()
	})
	return errs, filtered, nil
}
