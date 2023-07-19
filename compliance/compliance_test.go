// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"embed"
	"testing"

	"github.com/cucumber/godog"
)

//go:embed features/*
var featuresFS embed.FS

func TestSpecCompliance(t *testing.T) {
	paths := []string{"features"}
	checkFeaturesVersions(t, featuresFS, paths)

	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty,junit:report.xml",
			Paths:    paths,
			Tags:     versionsToTest(t),
			FS:       featuresFS,
			TestingT: t,
			Strict:   false,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func indexTemplateIncludes(arg1, arg2 string) error {
	return godog.ErrPending
}

func thePackageIsInstalled(packageName string) error {
	return godog.ErrPending
}

func thereIsAnIndexTemplateForPattern(arg1 string) error {
	return godog.ErrPending
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^index template "([^"]*)" includes "([^"]*)"$`, indexTemplateIncludes)
	ctx.Step(`^the "([^"]*)" package is installed$`, thePackageIsInstalled)
	ctx.Step(`^there is an index template for pattern "([^"]*)"$`, thereIsAnIndexTemplateForPattern)
}
