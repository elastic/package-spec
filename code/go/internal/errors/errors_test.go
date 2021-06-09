// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationErrorsMultiple(t *testing.T) {
	ve := ValidationErrors{}
	ve = append(ve, errors.New("error 1"))
	ve = append(ve, errors.New("error 2"))

	require.Len(t, ve, 2)
	require.Contains(t, ve.Error(), "found 2 validation errors:")
	require.Contains(t, ve.Error(), "error 1")
	require.Contains(t, ve.Error(), "error 2")
}

func TestValidationErrorsSingle(t *testing.T) {
	ve := ValidationErrors{}
	ve = append(ve, errors.New("error 1"))

	require.Len(t, ve, 1)
	require.Contains(t, ve.Error(), "found 1 validation error:")
	require.Contains(t, ve.Error(), "error 1")
}

func TestValidationErrorsNone(t *testing.T) {
	ve := ValidationErrors{}

	require.Len(t, ve, 0)
	require.Contains(t, ve.Error(), "found 0 validation errors")
}
