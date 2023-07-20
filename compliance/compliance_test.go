// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
		t.Fatalf("target deployment doesn't comply with Package Spec %s", versionToComply(t))
	}
}

func indexTemplateIncludes(arg1, arg2 string) error {
	return godog.ErrPending
}

func thePackageIsInstalled(packageName string) error {
	packagePath := filepath.Join("testdata", "packages", packageName)
	info, err := os.Stat(packagePath)
	if errors.Is(err, os.ErrNotExist) {
		return godog.ErrPending
	}
	if !info.IsDir() {
		return fmt.Errorf("\"%s\" should be a directory", packagePath)
	}

	elasticPackage, err := NewElasticPackage()
	if err != nil {
		return err
	}
	defer elasticPackage.Close()

	err = elasticPackage.Install(packagePath)
	if err != nil {
		return fmt.Errorf("cannot install package %q: %w", packagePath, err)
	}
	return nil
}

func aPolicyIsCreatedWithPackage(packageName string) error {
	const version = "1.0.0" // TODO: Add support for package and version

	kibana, err := NewKibanaClient()
	if err != nil {
		return err
	}
	_, err = kibana.CreatePolicyForPackage(packageName, version)
	if err != nil {
		return err
	}
	return nil
}

func aPolicyIsCreatedWithPackageAndDataset(packageName, dataset string) error {
	return godog.ErrPending
}

func thereIsAnIndexTemplateWithPattern(indexTemplateName, pattern string) error {
	return godog.ErrPending
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^index template "([^"]*)" includes "([^"]*)"$`, indexTemplateIncludes)
	ctx.Step(`^the "([^"]*)" package is installed$`, thePackageIsInstalled)
	ctx.Step(`^a policy is created with "([^"]*)" package$`, aPolicyIsCreatedWithPackage)
	ctx.Step(`^a policy is created with "([^"]*)" package and dataset "([^"]*)"$`, aPolicyIsCreatedWithPackageAndDataset)
	ctx.Step(`^there is an index template "([^"]*)" with pattern "([^"]*)"$`, thereIsAnIndexTemplateWithPattern)
}
