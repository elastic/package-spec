// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"os"

	"github.com/Masterminds/semver/v3"
)

const specVersionEnv = "TEST_SPEC_VERSION"

func versionToComply() (*semver.Version, error) {
	v := os.Getenv(specVersionEnv)
	if v == "" {
		return nil, fmt.Errorf("%s environment variable required with the version to test compliance", specVersionEnv)
	}

	return semver.NewVersion(v)
}
