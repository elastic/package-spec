// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package common

import (
	"os"
	"strconv"
)

// EnvVarWarningsAsErrors is the environment variable name used to enable warnings as errors
// this meachinsm will be removed once structured errors are supported https://github.com/elastic/package-spec/issues/342
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
