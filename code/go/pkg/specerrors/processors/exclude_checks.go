// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processors

import (
	"github.com/elastic/package-spec/v2/code/go/pkg/specerrors"
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
func (p ExcludeCheck) Process(issues specerrors.ValidationErrors) (ProcessResult, error) {
	if p.code == specerrors.UnassignedCode {
		return ProcessResult{Processed: issues, Removed: nil}, nil
	}

	errs, filtered := issues.Collect(func(i specerrors.ValidationError) bool {
		if i.Code() == specerrors.UnassignedCode {
			// Errors without assigned code cannot be skipped
			return true
		}
		return p.code != i.Code()
	})
	return ProcessResult{Processed: errs, Removed: filtered}, nil
}
