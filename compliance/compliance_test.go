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

func InitializeScenario(ctx *godog.ScenarioContext) {
	sm := newScenarioManager(ctx)

	ctx.Step(`^an "([^"]*)" package$`, sm.createPackage)
	ctx.Step(`^index template "([^"]*)"$`, sm.checkIndexTemplate)
	ctx.Step(`^the package has "([^"]*)"$`, sm.addToPackage)
	ctx.Step(`^the package is installed$`, sm.installPackage)
}
