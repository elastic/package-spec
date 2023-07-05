// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"embed"
	"fmt"
	"testing"

	"github.com/cucumber/godog"

	"github.com/Masterminds/semver/v3"
)

func indexTemplateForIncludesRuntimeFields(packageName string) error {
	//return godog.ErrPending
	return nil
}

func isCandidateToSupport(candidate, specVersion string) error {
	targetSpecVersion, err := semver.NewVersion(specVersion)
	if err != nil {
		return fmt.Errorf("failed to parse target version %s: %w", specVersion, err)
	}

	testVersion, err := versionToComply()
	if err != nil {
		return err
	}

	if targetSpecVersion.GreaterThan(testVersion) {
		return godog.ErrPending
	}

	return nil
}

func isInstalled(packageName string) error {
	//return godog.ErrPending
	return nil
}

//go:embed features/*
var featuresFS embed.FS

func TestSpecCompliance(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty,junit:report.xml",
			Paths:    []string{"features"},
			Tags:     "2.9.0", // XXX: Filtering of tests per version.
			FS:       featuresFS,
			TestingT: t,
			Strict:   false,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^index template for "([^"]*)" includes runtime fields\.$`, indexTemplateForIncludesRuntimeFields)
	ctx.Step(`^"([^"]*)" is candidate to support "([^"]*)"\.$`, isCandidateToSupport)
	ctx.Step(`^"([^"]*)" is installed\.$`, isInstalled)
}
