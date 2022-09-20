// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package common

import (
	"os"
	"strconv"
)

// EnvVarWarningsAsErrors is the environment variable name used to enable warnings as errors
// this meachinsm will be removed once structured errors are supported https://github.com/elastic/package-spec/v2/issues/342
const EnvVarWarningsAsErrors = "PACKAGE_SPEC_WARNINGS_AS_ERRORS"

// IsDefinedWarningsAsErrors checks whether or not warnings should be considered as errors,
// it checks the environment variable is defined and the value that it contains
func IsDefinedWarningsAsErrors() bool {
	var err error
	warningsAsErrors := false
	warningsAsErrorsStr, found := os.LookupEnv(EnvVarWarningsAsErrors)
	if found {
		warningsAsErrors, err = strconv.ParseBool(warningsAsErrorsStr)
		if err != nil {
			return false
		}
	}
	return warningsAsErrors
}

// EnableWarningsAsErrors is a function to enable warnings as errors, setting environment variable as true
func EnableWarningsAsErrors() error {
	if err := os.Setenv(EnvVarWarningsAsErrors, "true"); err != nil {
		return err
	}
	return nil
}

// DisableWarningsAsErrors is a function to disable warnings as errors, unsetting environment variable
func DisableWarningsAsErrors() error {
	if err := os.Unsetenv(EnvVarWarningsAsErrors); err != nil {
		return err
	}
	return nil
}
