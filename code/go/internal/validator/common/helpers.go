package common

import (
	"os"
	"strconv"
)

const EnvVarWarningsAsErrors = "PACKAGE_SPEC_WARNINGS_AS_ERRORS"

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

func EnableWarningsAsErrors() error {
	if err := os.Setenv(EnvVarWarningsAsErrors, "true"); err != nil {
		return err
	}
	return nil
}

func DisableWarningsAsErrors() error {
	if err := os.Unsetenv(EnvVarWarningsAsErrors); err != nil {
		return err
	}
	return nil
}
